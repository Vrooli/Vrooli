export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 7650,
          "end": 7654
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7655,
              "end": 7660
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7663,
                "end": 7668
              }
            },
            "loc": {
              "start": 7662,
              "end": 7668
            }
          },
          "loc": {
            "start": 7655,
            "end": 7668
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recommended",
              "loc": {
                "start": 7676,
                "end": 7687
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Resource_list",
                    "loc": {
                      "start": 7701,
                      "end": 7714
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7698,
                    "end": 7714
                  }
                }
              ],
              "loc": {
                "start": 7688,
                "end": 7720
              }
            },
            "loc": {
              "start": 7676,
              "end": 7720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 7725,
                "end": 7734
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Reminder_full",
                    "loc": {
                      "start": 7748,
                      "end": 7761
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7745,
                    "end": 7761
                  }
                }
              ],
              "loc": {
                "start": 7735,
                "end": 7767
              }
            },
            "loc": {
              "start": 7725,
              "end": 7767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 7772,
                "end": 7781
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Resource_list",
                    "loc": {
                      "start": 7795,
                      "end": 7808
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7792,
                    "end": 7808
                  }
                }
              ],
              "loc": {
                "start": 7782,
                "end": 7814
              }
            },
            "loc": {
              "start": 7772,
              "end": 7814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 7819,
                "end": 7828
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Schedule_list",
                    "loc": {
                      "start": 7842,
                      "end": 7855
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7839,
                    "end": 7855
                  }
                }
              ],
              "loc": {
                "start": 7829,
                "end": 7861
              }
            },
            "loc": {
              "start": 7819,
              "end": 7861
            }
          }
        ],
        "loc": {
          "start": 7670,
          "end": 7865
        }
      },
      "loc": {
        "start": 7650,
        "end": 7865
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 88,
                  "end": 100
                }
              },
              "loc": {
                "start": 88,
                "end": 100
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 114,
                      "end": 130
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 111,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 101,
                "end": 136
              }
            },
            "loc": {
              "start": 81,
              "end": 136
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 148,
                  "end": 152
                }
              },
              "loc": {
                "start": 148,
                "end": 152
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 166,
                      "end": 174
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 174
                  }
                }
              ],
              "loc": {
                "start": 153,
                "end": 180
              }
            },
            "loc": {
              "start": 141,
              "end": 180
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 182
        }
      },
      "loc": {
        "start": 69,
        "end": 182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 183,
          "end": 186
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 193,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 207,
                "end": 216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 216
            }
          }
        ],
        "loc": {
          "start": 187,
          "end": 218
        }
      },
      "loc": {
        "start": 183,
        "end": 218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 265,
          "end": 267
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 265,
        "end": 267
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 268,
          "end": 279
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 268,
        "end": 279
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 280,
          "end": 286
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 280,
        "end": 286
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 287,
          "end": 299
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 287,
        "end": 299
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 300,
          "end": 303
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 310,
                "end": 323
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 310,
              "end": 323
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 328,
                "end": 337
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 328,
              "end": 337
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 342,
                "end": 353
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 342,
              "end": 353
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 358,
                "end": 367
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 358,
              "end": 367
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 372,
                "end": 381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 372,
              "end": 381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 386,
                "end": 393
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 386,
              "end": 393
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 398,
                "end": 410
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 398,
              "end": 410
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 415,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 415,
              "end": 423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 428,
                "end": 442
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 453,
                      "end": 455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 453,
                    "end": 455
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 464,
                      "end": 474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 464,
                    "end": 474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 483,
                      "end": 493
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 483,
                    "end": 493
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 502,
                      "end": 509
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 502,
                    "end": 509
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 518,
                      "end": 529
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 529
                  }
                }
              ],
              "loc": {
                "start": 443,
                "end": 535
              }
            },
            "loc": {
              "start": 428,
              "end": 535
            }
          }
        ],
        "loc": {
          "start": 304,
          "end": 537
        }
      },
      "loc": {
        "start": 300,
        "end": 537
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 577,
          "end": 579
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 577,
        "end": 579
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 580,
          "end": 590
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 580,
        "end": 590
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 591,
          "end": 601
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 591,
        "end": 601
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 602,
          "end": 606
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 602,
        "end": 606
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "description",
        "loc": {
          "start": 607,
          "end": 618
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 607,
        "end": 618
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "dueDate",
        "loc": {
          "start": 619,
          "end": 626
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 619,
        "end": 626
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 627,
          "end": 632
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 627,
        "end": 632
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isComplete",
        "loc": {
          "start": 633,
          "end": 643
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 633,
        "end": 643
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderItems",
        "loc": {
          "start": 644,
          "end": 657
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 664,
                "end": 666
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 664,
              "end": 666
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 671,
                "end": 681
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 671,
              "end": 681
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 686,
                "end": 696
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 686,
              "end": 696
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 701,
                "end": 705
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 701,
              "end": 705
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 710,
                "end": 721
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 710,
              "end": 721
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 726,
                "end": 733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 726,
              "end": 733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 738,
                "end": 743
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 738,
              "end": 743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 748,
                "end": 758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 748,
              "end": 758
            }
          }
        ],
        "loc": {
          "start": 658,
          "end": 760
        }
      },
      "loc": {
        "start": 644,
        "end": 760
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderList",
        "loc": {
          "start": 761,
          "end": 773
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 780,
                "end": 782
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 780,
              "end": 782
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 787,
                "end": 797
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 787,
              "end": 797
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 802,
                "end": 812
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 802,
              "end": 812
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusMode",
              "loc": {
                "start": 817,
                "end": 826
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 837,
                      "end": 843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 858,
                            "end": 860
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 858,
                          "end": 860
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 873,
                            "end": 878
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 873,
                          "end": 878
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 891,
                            "end": 896
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 891,
                          "end": 896
                        }
                      }
                    ],
                    "loc": {
                      "start": 844,
                      "end": 906
                    }
                  },
                  "loc": {
                    "start": 837,
                    "end": 906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 915,
                      "end": 927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 942,
                            "end": 944
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 942,
                          "end": 944
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 957,
                            "end": 967
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 957,
                          "end": 967
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 980,
                            "end": 992
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1011,
                                  "end": 1013
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1011,
                                "end": 1013
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 1030,
                                  "end": 1038
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1030,
                                "end": 1038
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 1055,
                                  "end": 1066
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1055,
                                "end": 1066
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 1083,
                                  "end": 1087
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1083,
                                "end": 1087
                              }
                            }
                          ],
                          "loc": {
                            "start": 993,
                            "end": 1101
                          }
                        },
                        "loc": {
                          "start": 980,
                          "end": 1101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 1114,
                            "end": 1123
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1142,
                                  "end": 1144
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1142,
                                "end": 1144
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 1161,
                                  "end": 1166
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1161,
                                "end": 1166
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 1183,
                                  "end": 1187
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1183,
                                "end": 1187
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 1204,
                                  "end": 1211
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1204,
                                "end": 1211
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 1228,
                                  "end": 1240
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1263,
                                        "end": 1265
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1263,
                                      "end": 1265
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 1286,
                                        "end": 1294
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1286,
                                      "end": 1294
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 1315,
                                        "end": 1326
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1315,
                                      "end": 1326
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 1347,
                                        "end": 1351
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1347,
                                      "end": 1351
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1241,
                                  "end": 1369
                                }
                              },
                              "loc": {
                                "start": 1228,
                                "end": 1369
                              }
                            }
                          ],
                          "loc": {
                            "start": 1124,
                            "end": 1383
                          }
                        },
                        "loc": {
                          "start": 1114,
                          "end": 1383
                        }
                      }
                    ],
                    "loc": {
                      "start": 928,
                      "end": 1393
                    }
                  },
                  "loc": {
                    "start": 915,
                    "end": 1393
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 1402,
                      "end": 1410
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Schedule_common",
                          "loc": {
                            "start": 1428,
                            "end": 1443
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1425,
                          "end": 1443
                        }
                      }
                    ],
                    "loc": {
                      "start": 1411,
                      "end": 1453
                    }
                  },
                  "loc": {
                    "start": 1402,
                    "end": 1453
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1462,
                      "end": 1464
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1462,
                    "end": 1464
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1473,
                      "end": 1477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1473,
                    "end": 1477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1486,
                      "end": 1497
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1486,
                    "end": 1497
                  }
                }
              ],
              "loc": {
                "start": 827,
                "end": 1503
              }
            },
            "loc": {
              "start": 817,
              "end": 1503
            }
          }
        ],
        "loc": {
          "start": 774,
          "end": 1505
        }
      },
      "loc": {
        "start": 761,
        "end": 1505
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1545,
          "end": 1547
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1545,
        "end": 1547
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1548,
          "end": 1553
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1548,
        "end": 1553
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 1554,
          "end": 1558
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1554,
        "end": 1558
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 1559,
          "end": 1566
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1559,
        "end": 1566
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1567,
          "end": 1579
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1586,
                "end": 1588
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1586,
              "end": 1588
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1593,
                "end": 1601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1593,
              "end": 1601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1606,
                "end": 1617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1606,
              "end": 1617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1622,
                "end": 1626
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1622,
              "end": 1626
            }
          }
        ],
        "loc": {
          "start": 1580,
          "end": 1628
        }
      },
      "loc": {
        "start": 1567,
        "end": 1628
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1670,
          "end": 1672
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1670,
        "end": 1672
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1673,
          "end": 1683
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1673,
        "end": 1683
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1684,
          "end": 1694
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1684,
        "end": 1694
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 1695,
          "end": 1704
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1695,
        "end": 1704
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 1705,
          "end": 1712
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1705,
        "end": 1712
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 1713,
          "end": 1721
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1713,
        "end": 1721
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 1722,
          "end": 1732
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1739,
                "end": 1741
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1739,
              "end": 1741
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 1746,
                "end": 1763
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1746,
              "end": 1763
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 1768,
                "end": 1780
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1768,
              "end": 1780
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 1785,
                "end": 1795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1785,
              "end": 1795
            }
          }
        ],
        "loc": {
          "start": 1733,
          "end": 1797
        }
      },
      "loc": {
        "start": 1722,
        "end": 1797
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 1798,
          "end": 1809
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1816,
                "end": 1818
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1816,
              "end": 1818
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 1823,
                "end": 1837
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1823,
              "end": 1837
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 1842,
                "end": 1850
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1842,
              "end": 1850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 1855,
                "end": 1864
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1855,
              "end": 1864
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 1869,
                "end": 1879
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1869,
              "end": 1879
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 1884,
                "end": 1889
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1884,
              "end": 1889
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 1894,
                "end": 1901
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1894,
              "end": 1901
            }
          }
        ],
        "loc": {
          "start": 1810,
          "end": 1903
        }
      },
      "loc": {
        "start": 1798,
        "end": 1903
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1943,
          "end": 1949
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 1959,
                "end": 1969
              }
            },
            "directives": [],
            "loc": {
              "start": 1956,
              "end": 1969
            }
          }
        ],
        "loc": {
          "start": 1950,
          "end": 1971
        }
      },
      "loc": {
        "start": 1943,
        "end": 1971
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModes",
        "loc": {
          "start": 1972,
          "end": 1982
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1989,
                "end": 1995
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2006,
                      "end": 2008
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2006,
                    "end": 2008
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "color",
                    "loc": {
                      "start": 2017,
                      "end": 2022
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2017,
                    "end": 2022
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 2031,
                      "end": 2036
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2031,
                    "end": 2036
                  }
                }
              ],
              "loc": {
                "start": 1996,
                "end": 2042
              }
            },
            "loc": {
              "start": 1989,
              "end": 2042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 2047,
                "end": 2059
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2070,
                      "end": 2072
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2070,
                    "end": 2072
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2081,
                      "end": 2091
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2081,
                    "end": 2091
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2100,
                      "end": 2110
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2100,
                    "end": 2110
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminders",
                    "loc": {
                      "start": 2119,
                      "end": 2128
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2143,
                            "end": 2145
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2143,
                          "end": 2145
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2158,
                            "end": 2168
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2158,
                          "end": 2168
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2181,
                            "end": 2191
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2181,
                          "end": 2191
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2204,
                            "end": 2208
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2204,
                          "end": 2208
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2221,
                            "end": 2232
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2221,
                          "end": 2232
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dueDate",
                          "loc": {
                            "start": 2245,
                            "end": 2252
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2245,
                          "end": 2252
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 2265,
                            "end": 2270
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2265,
                          "end": 2270
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 2283,
                            "end": 2293
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2283,
                          "end": 2293
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderItems",
                          "loc": {
                            "start": 2306,
                            "end": 2319
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2338,
                                  "end": 2340
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2338,
                                "end": 2340
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2357,
                                  "end": 2367
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2357,
                                "end": 2367
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2384,
                                  "end": 2394
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2384,
                                "end": 2394
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2411,
                                  "end": 2415
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2411,
                                "end": 2415
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2432,
                                  "end": 2443
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2432,
                                "end": 2443
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 2460,
                                  "end": 2467
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2460,
                                "end": 2467
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2484,
                                  "end": 2489
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2484,
                                "end": 2489
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 2506,
                                  "end": 2516
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2506,
                                "end": 2516
                              }
                            }
                          ],
                          "loc": {
                            "start": 2320,
                            "end": 2530
                          }
                        },
                        "loc": {
                          "start": 2306,
                          "end": 2530
                        }
                      }
                    ],
                    "loc": {
                      "start": 2129,
                      "end": 2540
                    }
                  },
                  "loc": {
                    "start": 2119,
                    "end": 2540
                  }
                }
              ],
              "loc": {
                "start": 2060,
                "end": 2546
              }
            },
            "loc": {
              "start": 2047,
              "end": 2546
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resourceList",
              "loc": {
                "start": 2551,
                "end": 2563
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2574,
                      "end": 2576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2574,
                    "end": 2576
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2585,
                      "end": 2595
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2585,
                    "end": 2595
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2604,
                      "end": 2616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2631,
                            "end": 2633
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2631,
                          "end": 2633
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2646,
                            "end": 2654
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2646,
                          "end": 2654
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2667,
                            "end": 2678
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2667,
                          "end": 2678
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2691,
                            "end": 2695
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2691,
                          "end": 2695
                        }
                      }
                    ],
                    "loc": {
                      "start": 2617,
                      "end": 2705
                    }
                  },
                  "loc": {
                    "start": 2604,
                    "end": 2705
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resources",
                    "loc": {
                      "start": 2714,
                      "end": 2723
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2738,
                            "end": 2740
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2738,
                          "end": 2740
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 2753,
                            "end": 2758
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2753,
                          "end": 2758
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 2771,
                            "end": 2775
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2771,
                          "end": 2775
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "usedFor",
                          "loc": {
                            "start": 2788,
                            "end": 2795
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2788,
                          "end": 2795
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 2808,
                            "end": 2820
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2839,
                                  "end": 2841
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2839,
                                "end": 2841
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 2858,
                                  "end": 2866
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2858,
                                "end": 2866
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2883,
                                  "end": 2894
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2883,
                                "end": 2894
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2911,
                                  "end": 2915
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2911,
                                "end": 2915
                              }
                            }
                          ],
                          "loc": {
                            "start": 2821,
                            "end": 2929
                          }
                        },
                        "loc": {
                          "start": 2808,
                          "end": 2929
                        }
                      }
                    ],
                    "loc": {
                      "start": 2724,
                      "end": 2939
                    }
                  },
                  "loc": {
                    "start": 2714,
                    "end": 2939
                  }
                }
              ],
              "loc": {
                "start": 2564,
                "end": 2945
              }
            },
            "loc": {
              "start": 2551,
              "end": 2945
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2950,
                "end": 2952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2950,
              "end": 2952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2957,
                "end": 2961
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2957,
              "end": 2961
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2966,
                "end": 2977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2966,
              "end": 2977
            }
          }
        ],
        "loc": {
          "start": 1983,
          "end": 2979
        }
      },
      "loc": {
        "start": 1972,
        "end": 2979
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetings",
        "loc": {
          "start": 2980,
          "end": 2988
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2995,
                "end": 3001
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 3015,
                      "end": 3025
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3012,
                    "end": 3025
                  }
                }
              ],
              "loc": {
                "start": 3002,
                "end": 3031
              }
            },
            "loc": {
              "start": 2995,
              "end": 3031
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3036,
                "end": 3048
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3059,
                      "end": 3061
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3059,
                    "end": 3061
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3070,
                      "end": 3078
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3070,
                    "end": 3078
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3087,
                      "end": 3098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3087,
                    "end": 3098
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "link",
                    "loc": {
                      "start": 3107,
                      "end": 3111
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3107,
                    "end": 3111
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3120,
                      "end": 3124
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3120,
                    "end": 3124
                  }
                }
              ],
              "loc": {
                "start": 3049,
                "end": 3130
              }
            },
            "loc": {
              "start": 3036,
              "end": 3130
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3135,
                "end": 3137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3135,
              "end": 3137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3142,
                "end": 3152
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3142,
              "end": 3152
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3157,
                "end": 3167
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3157,
              "end": 3167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 3172,
                "end": 3194
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3172,
              "end": 3194
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnOrganizationProfile",
              "loc": {
                "start": 3199,
                "end": 3224
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3199,
              "end": 3224
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 3229,
                "end": 3241
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3252,
                      "end": 3254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3252,
                    "end": 3254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3263,
                      "end": 3274
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3263,
                    "end": 3274
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3283,
                      "end": 3289
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3283,
                    "end": 3289
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3298,
                      "end": 3310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3298,
                    "end": 3310
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 3319,
                      "end": 3322
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canAddMembers",
                          "loc": {
                            "start": 3337,
                            "end": 3350
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3337,
                          "end": 3350
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 3363,
                            "end": 3372
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3363,
                          "end": 3372
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 3385,
                            "end": 3396
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3385,
                          "end": 3396
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 3409,
                            "end": 3418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3409,
                          "end": 3418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 3431,
                            "end": 3440
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3431,
                          "end": 3440
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 3453,
                            "end": 3460
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3453,
                          "end": 3460
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 3473,
                            "end": 3485
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3473,
                          "end": 3485
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 3498,
                            "end": 3506
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3498,
                          "end": 3506
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 3519,
                            "end": 3533
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3552,
                                  "end": 3554
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3552,
                                "end": 3554
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3571,
                                  "end": 3581
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3571,
                                "end": 3581
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3598,
                                  "end": 3608
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3598,
                                "end": 3608
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3625,
                                  "end": 3632
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3625,
                                "end": 3632
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3649,
                                  "end": 3660
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3649,
                                "end": 3660
                              }
                            }
                          ],
                          "loc": {
                            "start": 3534,
                            "end": 3674
                          }
                        },
                        "loc": {
                          "start": 3519,
                          "end": 3674
                        }
                      }
                    ],
                    "loc": {
                      "start": 3323,
                      "end": 3684
                    }
                  },
                  "loc": {
                    "start": 3319,
                    "end": 3684
                  }
                }
              ],
              "loc": {
                "start": 3242,
                "end": 3690
              }
            },
            "loc": {
              "start": 3229,
              "end": 3690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 3695,
                "end": 3712
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "members",
                    "loc": {
                      "start": 3723,
                      "end": 3730
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3745,
                            "end": 3747
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3745,
                          "end": 3747
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3760,
                            "end": 3770
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3760,
                          "end": 3770
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3783,
                            "end": 3793
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3783,
                          "end": 3793
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3806,
                            "end": 3813
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3806,
                          "end": 3813
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3826,
                            "end": 3837
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3826,
                          "end": 3837
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 3850,
                            "end": 3855
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3874,
                                  "end": 3876
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3874,
                                "end": 3876
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3893,
                                  "end": 3903
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3893,
                                "end": 3903
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3920,
                                  "end": 3930
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3920,
                                "end": 3930
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3947,
                                  "end": 3951
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3947,
                                "end": 3951
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3968,
                                  "end": 3979
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3968,
                                "end": 3979
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 3996,
                                  "end": 4008
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3996,
                                "end": 4008
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "organization",
                                "loc": {
                                  "start": 4025,
                                  "end": 4037
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4060,
                                        "end": 4062
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4060,
                                      "end": 4062
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 4083,
                                        "end": 4094
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4083,
                                      "end": 4094
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 4115,
                                        "end": 4121
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4115,
                                      "end": 4121
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 4142,
                                        "end": 4154
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4142,
                                      "end": 4154
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 4175,
                                        "end": 4178
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canAddMembers",
                                            "loc": {
                                              "start": 4205,
                                              "end": 4218
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4205,
                                            "end": 4218
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 4243,
                                              "end": 4252
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4243,
                                            "end": 4252
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 4277,
                                              "end": 4288
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4277,
                                            "end": 4288
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 4313,
                                              "end": 4322
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4313,
                                            "end": 4322
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 4347,
                                              "end": 4356
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4347,
                                            "end": 4356
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 4381,
                                              "end": 4388
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4381,
                                            "end": 4388
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 4413,
                                              "end": 4425
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4413,
                                            "end": 4425
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 4450,
                                              "end": 4458
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4450,
                                            "end": 4458
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 4483,
                                              "end": 4497
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 4528,
                                                    "end": 4530
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4528,
                                                  "end": 4530
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 4559,
                                                    "end": 4569
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4559,
                                                  "end": 4569
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 4598,
                                                    "end": 4608
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4598,
                                                  "end": 4608
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 4637,
                                                    "end": 4644
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4637,
                                                  "end": 4644
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 4673,
                                                    "end": 4684
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4673,
                                                  "end": 4684
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4498,
                                              "end": 4710
                                            }
                                          },
                                          "loc": {
                                            "start": 4483,
                                            "end": 4710
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4179,
                                        "end": 4732
                                      }
                                    },
                                    "loc": {
                                      "start": 4175,
                                      "end": 4732
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4038,
                                  "end": 4750
                                }
                              },
                              "loc": {
                                "start": 4025,
                                "end": 4750
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4767,
                                  "end": 4779
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4802,
                                        "end": 4804
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4802,
                                      "end": 4804
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4825,
                                        "end": 4833
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4825,
                                      "end": 4833
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4854,
                                        "end": 4865
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4854,
                                      "end": 4865
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4780,
                                  "end": 4883
                                }
                              },
                              "loc": {
                                "start": 4767,
                                "end": 4883
                              }
                            }
                          ],
                          "loc": {
                            "start": 3856,
                            "end": 4897
                          }
                        },
                        "loc": {
                          "start": 3850,
                          "end": 4897
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 4910,
                            "end": 4913
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 4932,
                                  "end": 4941
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4932,
                                "end": 4941
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 4958,
                                  "end": 4967
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4958,
                                "end": 4967
                              }
                            }
                          ],
                          "loc": {
                            "start": 4914,
                            "end": 4981
                          }
                        },
                        "loc": {
                          "start": 4910,
                          "end": 4981
                        }
                      }
                    ],
                    "loc": {
                      "start": 3731,
                      "end": 4991
                    }
                  },
                  "loc": {
                    "start": 3723,
                    "end": 4991
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5000,
                      "end": 5002
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5000,
                    "end": 5002
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5011,
                      "end": 5021
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5011,
                    "end": 5021
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5030,
                      "end": 5040
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5030,
                    "end": 5040
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5049,
                      "end": 5053
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5049,
                    "end": 5053
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 5062,
                      "end": 5073
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5062,
                    "end": 5073
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 5082,
                      "end": 5094
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5082,
                    "end": 5094
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 5103,
                      "end": 5115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5130,
                            "end": 5132
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5130,
                          "end": 5132
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 5145,
                            "end": 5156
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5145,
                          "end": 5156
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 5169,
                            "end": 5175
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5169,
                          "end": 5175
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 5188,
                            "end": 5200
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5188,
                          "end": 5200
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 5213,
                            "end": 5216
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canAddMembers",
                                "loc": {
                                  "start": 5235,
                                  "end": 5248
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5235,
                                "end": 5248
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 5265,
                                  "end": 5274
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5265,
                                "end": 5274
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 5291,
                                  "end": 5302
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5291,
                                "end": 5302
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 5319,
                                  "end": 5328
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5319,
                                "end": 5328
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 5345,
                                  "end": 5354
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5345,
                                "end": 5354
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 5371,
                                  "end": 5378
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5371,
                                "end": 5378
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 5395,
                                  "end": 5407
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5395,
                                "end": 5407
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 5424,
                                  "end": 5432
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5424,
                                "end": 5432
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 5449,
                                  "end": 5463
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5486,
                                        "end": 5488
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5486,
                                      "end": 5488
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5509,
                                        "end": 5519
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5509,
                                      "end": 5519
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 5540,
                                        "end": 5550
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5540,
                                      "end": 5550
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 5571,
                                        "end": 5578
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5571,
                                      "end": 5578
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 5599,
                                        "end": 5610
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5599,
                                      "end": 5610
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5464,
                                  "end": 5628
                                }
                              },
                              "loc": {
                                "start": 5449,
                                "end": 5628
                              }
                            }
                          ],
                          "loc": {
                            "start": 5217,
                            "end": 5642
                          }
                        },
                        "loc": {
                          "start": 5213,
                          "end": 5642
                        }
                      }
                    ],
                    "loc": {
                      "start": 5116,
                      "end": 5652
                    }
                  },
                  "loc": {
                    "start": 5103,
                    "end": 5652
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5661,
                      "end": 5673
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5688,
                            "end": 5690
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5688,
                          "end": 5690
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5703,
                            "end": 5711
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5703,
                          "end": 5711
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5724,
                            "end": 5735
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5724,
                          "end": 5735
                        }
                      }
                    ],
                    "loc": {
                      "start": 5674,
                      "end": 5745
                    }
                  },
                  "loc": {
                    "start": 5661,
                    "end": 5745
                  }
                }
              ],
              "loc": {
                "start": 3713,
                "end": 5751
              }
            },
            "loc": {
              "start": 3695,
              "end": 5751
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 5756,
                "end": 5770
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5756,
              "end": 5770
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 5775,
                "end": 5787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5775,
              "end": 5787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5792,
                "end": 5795
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5806,
                      "end": 5815
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5806,
                    "end": 5815
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 5824,
                      "end": 5833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5824,
                    "end": 5833
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5842,
                      "end": 5851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5842,
                    "end": 5851
                  }
                }
              ],
              "loc": {
                "start": 5796,
                "end": 5857
              }
            },
            "loc": {
              "start": 5792,
              "end": 5857
            }
          }
        ],
        "loc": {
          "start": 2989,
          "end": 5859
        }
      },
      "loc": {
        "start": 2980,
        "end": 5859
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 5860,
          "end": 5871
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectVersion",
              "loc": {
                "start": 5878,
                "end": 5892
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5903,
                      "end": 5905
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5903,
                    "end": 5905
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 5914,
                      "end": 5924
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5914,
                    "end": 5924
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5933,
                      "end": 5941
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5933,
                    "end": 5941
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5950,
                      "end": 5959
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5950,
                    "end": 5959
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5968,
                      "end": 5980
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5968,
                    "end": 5980
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5989,
                      "end": 6001
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5989,
                    "end": 6001
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6010,
                      "end": 6014
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6029,
                            "end": 6031
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6029,
                          "end": 6031
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6044,
                            "end": 6053
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6044,
                          "end": 6053
                        }
                      }
                    ],
                    "loc": {
                      "start": 6015,
                      "end": 6063
                    }
                  },
                  "loc": {
                    "start": 6010,
                    "end": 6063
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6072,
                      "end": 6084
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6099,
                            "end": 6101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6099,
                          "end": 6101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6114,
                            "end": 6122
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6114,
                          "end": 6122
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6135,
                            "end": 6146
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6135,
                          "end": 6146
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6159,
                            "end": 6163
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6159,
                          "end": 6163
                        }
                      }
                    ],
                    "loc": {
                      "start": 6085,
                      "end": 6173
                    }
                  },
                  "loc": {
                    "start": 6072,
                    "end": 6173
                  }
                }
              ],
              "loc": {
                "start": 5893,
                "end": 6179
              }
            },
            "loc": {
              "start": 5878,
              "end": 6179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6184,
                "end": 6186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6184,
              "end": 6186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6191,
                "end": 6200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6191,
              "end": 6200
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6205,
                "end": 6224
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6205,
              "end": 6224
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6229,
                "end": 6244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6229,
              "end": 6244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 6249,
                "end": 6258
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6249,
              "end": 6258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 6263,
                "end": 6274
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6263,
              "end": 6274
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 6279,
                "end": 6290
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6279,
              "end": 6290
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6295,
                "end": 6299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6295,
              "end": 6299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 6304,
                "end": 6310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6304,
              "end": 6310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 6315,
                "end": 6325
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6315,
              "end": 6325
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 6330,
                "end": 6342
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6356,
                      "end": 6372
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6353,
                    "end": 6372
                  }
                }
              ],
              "loc": {
                "start": 6343,
                "end": 6378
              }
            },
            "loc": {
              "start": 6330,
              "end": 6378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 6383,
                "end": 6387
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 6401,
                      "end": 6409
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6398,
                    "end": 6409
                  }
                }
              ],
              "loc": {
                "start": 6388,
                "end": 6415
              }
            },
            "loc": {
              "start": 6383,
              "end": 6415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6420,
                "end": 6423
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6434,
                      "end": 6443
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6434,
                    "end": 6443
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6452,
                      "end": 6461
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6452,
                    "end": 6461
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6470,
                      "end": 6477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6470,
                    "end": 6477
                  }
                }
              ],
              "loc": {
                "start": 6424,
                "end": 6483
              }
            },
            "loc": {
              "start": 6420,
              "end": 6483
            }
          }
        ],
        "loc": {
          "start": 5872,
          "end": 6485
        }
      },
      "loc": {
        "start": 5860,
        "end": 6485
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 6486,
          "end": 6497
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineVersion",
              "loc": {
                "start": 6504,
                "end": 6518
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6529,
                      "end": 6531
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6529,
                    "end": 6531
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 6540,
                      "end": 6550
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6540,
                    "end": 6550
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 6559,
                      "end": 6572
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6559,
                    "end": 6572
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6581,
                      "end": 6591
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6581,
                    "end": 6591
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 6600,
                      "end": 6609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6600,
                    "end": 6609
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6618,
                      "end": 6626
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6618,
                    "end": 6626
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6635,
                      "end": 6644
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6635,
                    "end": 6644
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6653,
                      "end": 6657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6672,
                            "end": 6674
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6672,
                          "end": 6674
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 6687,
                            "end": 6697
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6687,
                          "end": 6697
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6710,
                            "end": 6719
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6710,
                          "end": 6719
                        }
                      }
                    ],
                    "loc": {
                      "start": 6658,
                      "end": 6729
                    }
                  },
                  "loc": {
                    "start": 6653,
                    "end": 6729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6738,
                      "end": 6750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6765,
                            "end": 6767
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6765,
                          "end": 6767
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6780,
                            "end": 6788
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6780,
                          "end": 6788
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6801,
                            "end": 6812
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6801,
                          "end": 6812
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 6825,
                            "end": 6837
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6825,
                          "end": 6837
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6850,
                            "end": 6854
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6850,
                          "end": 6854
                        }
                      }
                    ],
                    "loc": {
                      "start": 6751,
                      "end": 6864
                    }
                  },
                  "loc": {
                    "start": 6738,
                    "end": 6864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6873,
                      "end": 6885
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6873,
                    "end": 6885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6894,
                      "end": 6906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6894,
                    "end": 6906
                  }
                }
              ],
              "loc": {
                "start": 6519,
                "end": 6912
              }
            },
            "loc": {
              "start": 6504,
              "end": 6912
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6917,
                "end": 6919
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6917,
              "end": 6919
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6924,
                "end": 6933
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6924,
              "end": 6933
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6938,
                "end": 6957
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6938,
              "end": 6957
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6962,
                "end": 6977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6962,
              "end": 6977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 6982,
                "end": 6991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6982,
              "end": 6991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 6996,
                "end": 7007
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6996,
              "end": 7007
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7012,
                "end": 7023
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7012,
              "end": 7023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7028,
                "end": 7032
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7028,
              "end": 7032
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7037,
                "end": 7043
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7037,
              "end": 7043
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7048,
                "end": 7058
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7048,
              "end": 7058
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 7063,
                "end": 7074
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7063,
              "end": 7074
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 7079,
                "end": 7098
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7079,
              "end": 7098
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 7103,
                "end": 7115
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 7129,
                      "end": 7145
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7126,
                    "end": 7145
                  }
                }
              ],
              "loc": {
                "start": 7116,
                "end": 7151
              }
            },
            "loc": {
              "start": 7103,
              "end": 7151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 7156,
                "end": 7160
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 7174,
                      "end": 7182
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7171,
                    "end": 7182
                  }
                }
              ],
              "loc": {
                "start": 7161,
                "end": 7188
              }
            },
            "loc": {
              "start": 7156,
              "end": 7188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7193,
                "end": 7196
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7207,
                      "end": 7216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7207,
                    "end": 7216
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7225,
                      "end": 7234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7225,
                    "end": 7234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7243,
                      "end": 7250
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7243,
                    "end": 7250
                  }
                }
              ],
              "loc": {
                "start": 7197,
                "end": 7256
              }
            },
            "loc": {
              "start": 7193,
              "end": 7256
            }
          }
        ],
        "loc": {
          "start": 6498,
          "end": 7258
        }
      },
      "loc": {
        "start": 6486,
        "end": 7258
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7259,
          "end": 7261
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7259,
        "end": 7261
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7262,
          "end": 7272
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7262,
        "end": 7272
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7273,
          "end": 7283
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7273,
        "end": 7283
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 7284,
          "end": 7293
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7284,
        "end": 7293
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 7294,
          "end": 7301
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7294,
        "end": 7301
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 7302,
          "end": 7310
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7302,
        "end": 7310
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 7311,
          "end": 7321
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7328,
                "end": 7330
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7328,
              "end": 7330
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 7335,
                "end": 7352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7335,
              "end": 7352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 7357,
                "end": 7369
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7357,
              "end": 7369
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 7374,
                "end": 7384
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7374,
              "end": 7384
            }
          }
        ],
        "loc": {
          "start": 7322,
          "end": 7386
        }
      },
      "loc": {
        "start": 7311,
        "end": 7386
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 7387,
          "end": 7398
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7405,
                "end": 7407
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7405,
              "end": 7407
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 7412,
                "end": 7426
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7412,
              "end": 7426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 7431,
                "end": 7439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7431,
              "end": 7439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 7444,
                "end": 7453
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7444,
              "end": 7453
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 7458,
                "end": 7468
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7458,
              "end": 7468
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 7473,
                "end": 7478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7473,
              "end": 7478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 7483,
                "end": 7490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7483,
              "end": 7490
            }
          }
        ],
        "loc": {
          "start": 7399,
          "end": 7492
        }
      },
      "loc": {
        "start": 7387,
        "end": 7492
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7523,
          "end": 7525
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7523,
        "end": 7525
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7526,
          "end": 7536
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7526,
        "end": 7536
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7537,
          "end": 7547
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7537,
        "end": 7547
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7548,
          "end": 7559
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7548,
        "end": 7559
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7560,
          "end": 7566
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7560,
        "end": 7566
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7567,
          "end": 7572
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7567,
        "end": 7572
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7573,
          "end": 7593
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7573,
        "end": 7593
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7594,
          "end": 7598
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7594,
        "end": 7598
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7599,
          "end": 7611
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7599,
        "end": 7611
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 88,
                        "end": 100
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 100
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 114,
                            "end": 130
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 111,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 101,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 136
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 148,
                        "end": 152
                      }
                    },
                    "loc": {
                      "start": 148,
                      "end": 152
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 166,
                            "end": 174
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 163,
                          "end": 174
                        }
                      }
                    ],
                    "loc": {
                      "start": 153,
                      "end": 180
                    }
                  },
                  "loc": {
                    "start": 141,
                    "end": 180
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 182
              }
            },
            "loc": {
              "start": 69,
              "end": 182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 183,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 193,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 207,
                      "end": 216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 216
                  }
                }
              ],
              "loc": {
                "start": 187,
                "end": 218
              }
            },
            "loc": {
              "start": 183,
              "end": 218
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 220
        }
      },
      "loc": {
        "start": 1,
        "end": 220
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 230,
          "end": 246
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 250,
            "end": 262
          }
        },
        "loc": {
          "start": 250,
          "end": 262
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 265,
                "end": 267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 265,
              "end": 267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 268,
                "end": 279
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 268,
              "end": 279
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 280,
                "end": 286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 280,
              "end": 286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 287,
                "end": 299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 287,
              "end": 299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 300,
                "end": 303
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 310,
                      "end": 323
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 310,
                    "end": 323
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 328,
                      "end": 337
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 328,
                    "end": 337
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 342,
                      "end": 353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 342,
                    "end": 353
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 358,
                      "end": 367
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 358,
                    "end": 367
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 372,
                      "end": 381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 372,
                    "end": 381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 386,
                      "end": 393
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 386,
                    "end": 393
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 398,
                      "end": 410
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 398,
                    "end": 410
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 415,
                      "end": 423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 415,
                    "end": 423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 428,
                      "end": 442
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 453,
                            "end": 455
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 453,
                          "end": 455
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 464,
                            "end": 474
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 464,
                          "end": 474
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 483,
                            "end": 493
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 483,
                          "end": 493
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 502,
                            "end": 509
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 502,
                          "end": 509
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 518,
                            "end": 529
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 518,
                          "end": 529
                        }
                      }
                    ],
                    "loc": {
                      "start": 443,
                      "end": 535
                    }
                  },
                  "loc": {
                    "start": 428,
                    "end": 535
                  }
                }
              ],
              "loc": {
                "start": 304,
                "end": 537
              }
            },
            "loc": {
              "start": 300,
              "end": 537
            }
          }
        ],
        "loc": {
          "start": 263,
          "end": 539
        }
      },
      "loc": {
        "start": 221,
        "end": 539
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 549,
          "end": 562
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 566,
            "end": 574
          }
        },
        "loc": {
          "start": 566,
          "end": 574
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 577,
                "end": 579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 577,
              "end": 579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 580,
                "end": 590
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 580,
              "end": 590
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 591,
                "end": 601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 591,
              "end": 601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 602,
                "end": 606
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 602,
              "end": 606
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 607,
                "end": 618
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 607,
              "end": 618
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 619,
                "end": 626
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 619,
              "end": 626
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 627,
                "end": 632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 627,
              "end": 632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 633,
                "end": 643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 633,
              "end": 643
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 644,
                "end": 657
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 664,
                      "end": 666
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 664,
                    "end": 666
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 671,
                      "end": 681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 671,
                    "end": 681
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 686,
                      "end": 696
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 686,
                    "end": 696
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 701,
                      "end": 705
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 701,
                    "end": 705
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 710,
                      "end": 721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 710,
                    "end": 721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 726,
                      "end": 733
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 726,
                    "end": 733
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 738,
                      "end": 743
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 738,
                    "end": 743
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 748,
                      "end": 758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 748,
                    "end": 758
                  }
                }
              ],
              "loc": {
                "start": 658,
                "end": 760
              }
            },
            "loc": {
              "start": 644,
              "end": 760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 761,
                "end": 773
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 780,
                      "end": 782
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 780,
                    "end": 782
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 787,
                      "end": 797
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 787,
                    "end": 797
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 802,
                      "end": 812
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 802,
                    "end": 812
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 817,
                      "end": 826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 837,
                            "end": 843
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 858,
                                  "end": 860
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 858,
                                "end": 860
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 873,
                                  "end": 878
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 873,
                                "end": 878
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 891,
                                  "end": 896
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 891,
                                "end": 896
                              }
                            }
                          ],
                          "loc": {
                            "start": 844,
                            "end": 906
                          }
                        },
                        "loc": {
                          "start": 837,
                          "end": 906
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 915,
                            "end": 927
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 942,
                                  "end": 944
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 942,
                                "end": 944
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 957,
                                  "end": 967
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 957,
                                "end": 967
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 980,
                                  "end": 992
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1011,
                                        "end": 1013
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1011,
                                      "end": 1013
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 1030,
                                        "end": 1038
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1030,
                                      "end": 1038
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 1055,
                                        "end": 1066
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1055,
                                      "end": 1066
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 1083,
                                        "end": 1087
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1083,
                                      "end": 1087
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 993,
                                  "end": 1101
                                }
                              },
                              "loc": {
                                "start": 980,
                                "end": 1101
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 1114,
                                  "end": 1123
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1142,
                                        "end": 1144
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1142,
                                      "end": 1144
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 1161,
                                        "end": 1166
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1161,
                                      "end": 1166
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 1183,
                                        "end": 1187
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1183,
                                      "end": 1187
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 1204,
                                        "end": 1211
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1204,
                                      "end": 1211
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 1228,
                                        "end": 1240
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1263,
                                              "end": 1265
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1263,
                                            "end": 1265
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 1286,
                                              "end": 1294
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1286,
                                            "end": 1294
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1315,
                                              "end": 1326
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1315,
                                            "end": 1326
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1347,
                                              "end": 1351
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1347,
                                            "end": 1351
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1241,
                                        "end": 1369
                                      }
                                    },
                                    "loc": {
                                      "start": 1228,
                                      "end": 1369
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1124,
                                  "end": 1383
                                }
                              },
                              "loc": {
                                "start": 1114,
                                "end": 1383
                              }
                            }
                          ],
                          "loc": {
                            "start": 928,
                            "end": 1393
                          }
                        },
                        "loc": {
                          "start": 915,
                          "end": 1393
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 1402,
                            "end": 1410
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 1428,
                                  "end": 1443
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1425,
                                "end": 1443
                              }
                            }
                          ],
                          "loc": {
                            "start": 1411,
                            "end": 1453
                          }
                        },
                        "loc": {
                          "start": 1402,
                          "end": 1453
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1462,
                            "end": 1464
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1462,
                          "end": 1464
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1473,
                            "end": 1477
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1473,
                          "end": 1477
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1486,
                            "end": 1497
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1486,
                          "end": 1497
                        }
                      }
                    ],
                    "loc": {
                      "start": 827,
                      "end": 1503
                    }
                  },
                  "loc": {
                    "start": 817,
                    "end": 1503
                  }
                }
              ],
              "loc": {
                "start": 774,
                "end": 1505
              }
            },
            "loc": {
              "start": 761,
              "end": 1505
            }
          }
        ],
        "loc": {
          "start": 575,
          "end": 1507
        }
      },
      "loc": {
        "start": 540,
        "end": 1507
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1517,
          "end": 1530
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1534,
            "end": 1542
          }
        },
        "loc": {
          "start": 1534,
          "end": 1542
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1545,
                "end": 1547
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1545,
              "end": 1547
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1548,
                "end": 1553
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1548,
              "end": 1553
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1554,
                "end": 1558
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1554,
              "end": 1558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1559,
                "end": 1566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1559,
              "end": 1566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1567,
                "end": 1579
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1586,
                      "end": 1588
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1586,
                    "end": 1588
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1593,
                      "end": 1601
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1593,
                    "end": 1601
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1606,
                      "end": 1617
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1606,
                    "end": 1617
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1622,
                      "end": 1626
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1622,
                    "end": 1626
                  }
                }
              ],
              "loc": {
                "start": 1580,
                "end": 1628
              }
            },
            "loc": {
              "start": 1567,
              "end": 1628
            }
          }
        ],
        "loc": {
          "start": 1543,
          "end": 1630
        }
      },
      "loc": {
        "start": 1508,
        "end": 1630
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 1640,
          "end": 1655
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1659,
            "end": 1667
          }
        },
        "loc": {
          "start": 1659,
          "end": 1667
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1670,
                "end": 1672
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1670,
              "end": 1672
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1673,
                "end": 1683
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1673,
              "end": 1683
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1684,
                "end": 1694
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1684,
              "end": 1694
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1695,
                "end": 1704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1695,
              "end": 1704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 1705,
                "end": 1712
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1705,
              "end": 1712
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 1713,
                "end": 1721
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1713,
              "end": 1721
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 1722,
                "end": 1732
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1739,
                      "end": 1741
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1739,
                    "end": 1741
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 1746,
                      "end": 1763
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1746,
                    "end": 1763
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 1768,
                      "end": 1780
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1768,
                    "end": 1780
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 1785,
                      "end": 1795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1785,
                    "end": 1795
                  }
                }
              ],
              "loc": {
                "start": 1733,
                "end": 1797
              }
            },
            "loc": {
              "start": 1722,
              "end": 1797
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 1798,
                "end": 1809
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1816,
                      "end": 1818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1816,
                    "end": 1818
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 1823,
                      "end": 1837
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1823,
                    "end": 1837
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 1842,
                      "end": 1850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1842,
                    "end": 1850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 1855,
                      "end": 1864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1855,
                    "end": 1864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 1869,
                      "end": 1879
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1869,
                    "end": 1879
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 1884,
                      "end": 1889
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1884,
                    "end": 1889
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 1894,
                      "end": 1901
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1894,
                    "end": 1901
                  }
                }
              ],
              "loc": {
                "start": 1810,
                "end": 1903
              }
            },
            "loc": {
              "start": 1798,
              "end": 1903
            }
          }
        ],
        "loc": {
          "start": 1668,
          "end": 1905
        }
      },
      "loc": {
        "start": 1631,
        "end": 1905
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 1915,
          "end": 1928
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1932,
            "end": 1940
          }
        },
        "loc": {
          "start": 1932,
          "end": 1940
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1943,
                "end": 1949
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1959,
                      "end": 1969
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1956,
                    "end": 1969
                  }
                }
              ],
              "loc": {
                "start": 1950,
                "end": 1971
              }
            },
            "loc": {
              "start": 1943,
              "end": 1971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 1972,
                "end": 1982
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 1989,
                      "end": 1995
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2006,
                            "end": 2008
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2006,
                          "end": 2008
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 2017,
                            "end": 2022
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2017,
                          "end": 2022
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2031,
                            "end": 2036
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2031,
                          "end": 2036
                        }
                      }
                    ],
                    "loc": {
                      "start": 1996,
                      "end": 2042
                    }
                  },
                  "loc": {
                    "start": 1989,
                    "end": 2042
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminderList",
                    "loc": {
                      "start": 2047,
                      "end": 2059
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2070,
                            "end": 2072
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2070,
                          "end": 2072
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2081,
                            "end": 2091
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2081,
                          "end": 2091
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2100,
                            "end": 2110
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2100,
                          "end": 2110
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminders",
                          "loc": {
                            "start": 2119,
                            "end": 2128
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2143,
                                  "end": 2145
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2143,
                                "end": 2145
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2158,
                                  "end": 2168
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2158,
                                "end": 2168
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2181,
                                  "end": 2191
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2181,
                                "end": 2191
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2204,
                                  "end": 2208
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2204,
                                "end": 2208
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2221,
                                  "end": 2232
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2221,
                                "end": 2232
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 2245,
                                  "end": 2252
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2245,
                                "end": 2252
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2265,
                                  "end": 2270
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2265,
                                "end": 2270
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 2283,
                                  "end": 2293
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2283,
                                "end": 2293
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderItems",
                                "loc": {
                                  "start": 2306,
                                  "end": 2319
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2338,
                                        "end": 2340
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2338,
                                      "end": 2340
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2357,
                                        "end": 2367
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2357,
                                      "end": 2367
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 2384,
                                        "end": 2394
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2384,
                                      "end": 2394
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2411,
                                        "end": 2415
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2411,
                                      "end": 2415
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2432,
                                        "end": 2443
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2432,
                                      "end": 2443
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 2460,
                                        "end": 2467
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2460,
                                      "end": 2467
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 2484,
                                        "end": 2489
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2484,
                                      "end": 2489
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 2506,
                                        "end": 2516
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2506,
                                      "end": 2516
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2320,
                                  "end": 2530
                                }
                              },
                              "loc": {
                                "start": 2306,
                                "end": 2530
                              }
                            }
                          ],
                          "loc": {
                            "start": 2129,
                            "end": 2540
                          }
                        },
                        "loc": {
                          "start": 2119,
                          "end": 2540
                        }
                      }
                    ],
                    "loc": {
                      "start": 2060,
                      "end": 2546
                    }
                  },
                  "loc": {
                    "start": 2047,
                    "end": 2546
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 2551,
                      "end": 2563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2574,
                            "end": 2576
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2574,
                          "end": 2576
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2585,
                            "end": 2595
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2585,
                          "end": 2595
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 2604,
                            "end": 2616
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2631,
                                  "end": 2633
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2631,
                                "end": 2633
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 2646,
                                  "end": 2654
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2646,
                                "end": 2654
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2667,
                                  "end": 2678
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2667,
                                "end": 2678
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2691,
                                  "end": 2695
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2691,
                                "end": 2695
                              }
                            }
                          ],
                          "loc": {
                            "start": 2617,
                            "end": 2705
                          }
                        },
                        "loc": {
                          "start": 2604,
                          "end": 2705
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 2714,
                            "end": 2723
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2738,
                                  "end": 2740
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2738,
                                "end": 2740
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2753,
                                  "end": 2758
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2753,
                                "end": 2758
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 2771,
                                  "end": 2775
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2771,
                                "end": 2775
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 2788,
                                  "end": 2795
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2788,
                                "end": 2795
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 2808,
                                  "end": 2820
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2839,
                                        "end": 2841
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2839,
                                      "end": 2841
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 2858,
                                        "end": 2866
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2858,
                                      "end": 2866
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2883,
                                        "end": 2894
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2883,
                                      "end": 2894
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2911,
                                        "end": 2915
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2911,
                                      "end": 2915
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2821,
                                  "end": 2929
                                }
                              },
                              "loc": {
                                "start": 2808,
                                "end": 2929
                              }
                            }
                          ],
                          "loc": {
                            "start": 2724,
                            "end": 2939
                          }
                        },
                        "loc": {
                          "start": 2714,
                          "end": 2939
                        }
                      }
                    ],
                    "loc": {
                      "start": 2564,
                      "end": 2945
                    }
                  },
                  "loc": {
                    "start": 2551,
                    "end": 2945
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2950,
                      "end": 2952
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2950,
                    "end": 2952
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2957,
                      "end": 2961
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2957,
                    "end": 2961
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2966,
                      "end": 2977
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2966,
                    "end": 2977
                  }
                }
              ],
              "loc": {
                "start": 1983,
                "end": 2979
              }
            },
            "loc": {
              "start": 1972,
              "end": 2979
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetings",
              "loc": {
                "start": 2980,
                "end": 2988
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 2995,
                      "end": 3001
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Label_list",
                          "loc": {
                            "start": 3015,
                            "end": 3025
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3012,
                          "end": 3025
                        }
                      }
                    ],
                    "loc": {
                      "start": 3002,
                      "end": 3031
                    }
                  },
                  "loc": {
                    "start": 2995,
                    "end": 3031
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 3036,
                      "end": 3048
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3059,
                            "end": 3061
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3059,
                          "end": 3061
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3070,
                            "end": 3078
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3070,
                          "end": 3078
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3087,
                            "end": 3098
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3087,
                          "end": 3098
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 3107,
                            "end": 3111
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3107,
                          "end": 3111
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3120,
                            "end": 3124
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3120,
                          "end": 3124
                        }
                      }
                    ],
                    "loc": {
                      "start": 3049,
                      "end": 3130
                    }
                  },
                  "loc": {
                    "start": 3036,
                    "end": 3130
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3135,
                      "end": 3137
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3135,
                    "end": 3137
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3142,
                      "end": 3152
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3142,
                    "end": 3152
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3157,
                      "end": 3167
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3157,
                    "end": 3167
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 3172,
                      "end": 3194
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3172,
                    "end": 3194
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnOrganizationProfile",
                    "loc": {
                      "start": 3199,
                      "end": 3224
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3199,
                    "end": 3224
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 3229,
                      "end": 3241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3252,
                            "end": 3254
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3252,
                          "end": 3254
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 3263,
                            "end": 3274
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3263,
                          "end": 3274
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 3283,
                            "end": 3289
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3283,
                          "end": 3289
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 3298,
                            "end": 3310
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3298,
                          "end": 3310
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 3319,
                            "end": 3322
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canAddMembers",
                                "loc": {
                                  "start": 3337,
                                  "end": 3350
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3337,
                                "end": 3350
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 3363,
                                  "end": 3372
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3363,
                                "end": 3372
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 3385,
                                  "end": 3396
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3385,
                                "end": 3396
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 3409,
                                  "end": 3418
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3409,
                                "end": 3418
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 3431,
                                  "end": 3440
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3431,
                                "end": 3440
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 3453,
                                  "end": 3460
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3453,
                                "end": 3460
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 3473,
                                  "end": 3485
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3473,
                                "end": 3485
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 3498,
                                  "end": 3506
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3498,
                                "end": 3506
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 3519,
                                  "end": 3533
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3552,
                                        "end": 3554
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3552,
                                      "end": 3554
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3571,
                                        "end": 3581
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3571,
                                      "end": 3581
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3598,
                                        "end": 3608
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3598,
                                      "end": 3608
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 3625,
                                        "end": 3632
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3625,
                                      "end": 3632
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3649,
                                        "end": 3660
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3649,
                                      "end": 3660
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3534,
                                  "end": 3674
                                }
                              },
                              "loc": {
                                "start": 3519,
                                "end": 3674
                              }
                            }
                          ],
                          "loc": {
                            "start": 3323,
                            "end": 3684
                          }
                        },
                        "loc": {
                          "start": 3319,
                          "end": 3684
                        }
                      }
                    ],
                    "loc": {
                      "start": 3242,
                      "end": 3690
                    }
                  },
                  "loc": {
                    "start": 3229,
                    "end": 3690
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 3695,
                      "end": 3712
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "members",
                          "loc": {
                            "start": 3723,
                            "end": 3730
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3745,
                                  "end": 3747
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3745,
                                "end": 3747
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3760,
                                  "end": 3770
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3760,
                                "end": 3770
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3783,
                                  "end": 3793
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3783,
                                "end": 3793
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3806,
                                  "end": 3813
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3806,
                                "end": 3813
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3826,
                                  "end": 3837
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3826,
                                "end": 3837
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 3850,
                                  "end": 3855
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3874,
                                        "end": 3876
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3874,
                                      "end": 3876
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3893,
                                        "end": 3903
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3893,
                                      "end": 3903
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3920,
                                        "end": 3930
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3920,
                                      "end": 3930
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3947,
                                        "end": 3951
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3947,
                                      "end": 3951
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3968,
                                        "end": 3979
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3968,
                                      "end": 3979
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 3996,
                                        "end": 4008
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3996,
                                      "end": 4008
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "organization",
                                      "loc": {
                                        "start": 4025,
                                        "end": 4037
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4060,
                                              "end": 4062
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4060,
                                            "end": 4062
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 4083,
                                              "end": 4094
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4083,
                                            "end": 4094
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 4115,
                                              "end": 4121
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4115,
                                            "end": 4121
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 4142,
                                              "end": 4154
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4142,
                                            "end": 4154
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 4175,
                                              "end": 4178
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canAddMembers",
                                                  "loc": {
                                                    "start": 4205,
                                                    "end": 4218
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4205,
                                                  "end": 4218
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 4243,
                                                    "end": 4252
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4243,
                                                  "end": 4252
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 4277,
                                                    "end": 4288
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4277,
                                                  "end": 4288
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 4313,
                                                    "end": 4322
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4313,
                                                  "end": 4322
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 4347,
                                                    "end": 4356
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4347,
                                                  "end": 4356
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 4381,
                                                    "end": 4388
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4381,
                                                  "end": 4388
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 4413,
                                                    "end": 4425
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4413,
                                                  "end": 4425
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 4450,
                                                    "end": 4458
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4450,
                                                  "end": 4458
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 4483,
                                                    "end": 4497
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 4528,
                                                          "end": 4530
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4528,
                                                        "end": 4530
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 4559,
                                                          "end": 4569
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4559,
                                                        "end": 4569
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 4598,
                                                          "end": 4608
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4598,
                                                        "end": 4608
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 4637,
                                                          "end": 4644
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4637,
                                                        "end": 4644
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 4673,
                                                          "end": 4684
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4673,
                                                        "end": 4684
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 4498,
                                                    "end": 4710
                                                  }
                                                },
                                                "loc": {
                                                  "start": 4483,
                                                  "end": 4710
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4179,
                                              "end": 4732
                                            }
                                          },
                                          "loc": {
                                            "start": 4175,
                                            "end": 4732
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4038,
                                        "end": 4750
                                      }
                                    },
                                    "loc": {
                                      "start": 4025,
                                      "end": 4750
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4767,
                                        "end": 4779
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4802,
                                              "end": 4804
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4802,
                                            "end": 4804
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4825,
                                              "end": 4833
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4825,
                                            "end": 4833
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4854,
                                              "end": 4865
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4854,
                                            "end": 4865
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4780,
                                        "end": 4883
                                      }
                                    },
                                    "loc": {
                                      "start": 4767,
                                      "end": 4883
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3856,
                                  "end": 4897
                                }
                              },
                              "loc": {
                                "start": 3850,
                                "end": 4897
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 4910,
                                  "end": 4913
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 4932,
                                        "end": 4941
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4932,
                                      "end": 4941
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 4958,
                                        "end": 4967
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4958,
                                      "end": 4967
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4914,
                                  "end": 4981
                                }
                              },
                              "loc": {
                                "start": 4910,
                                "end": 4981
                              }
                            }
                          ],
                          "loc": {
                            "start": 3731,
                            "end": 4991
                          }
                        },
                        "loc": {
                          "start": 3723,
                          "end": 4991
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5000,
                            "end": 5002
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5000,
                          "end": 5002
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5011,
                            "end": 5021
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5011,
                          "end": 5021
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5030,
                            "end": 5040
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5030,
                          "end": 5040
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5049,
                            "end": 5053
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5049,
                          "end": 5053
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 5062,
                            "end": 5073
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5062,
                          "end": 5073
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 5082,
                            "end": 5094
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5082,
                          "end": 5094
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "organization",
                          "loc": {
                            "start": 5103,
                            "end": 5115
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5130,
                                  "end": 5132
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5130,
                                "end": 5132
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 5145,
                                  "end": 5156
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5145,
                                "end": 5156
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 5169,
                                  "end": 5175
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5169,
                                "end": 5175
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 5188,
                                  "end": 5200
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5188,
                                "end": 5200
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5213,
                                  "end": 5216
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canAddMembers",
                                      "loc": {
                                        "start": 5235,
                                        "end": 5248
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5235,
                                      "end": 5248
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 5265,
                                        "end": 5274
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5265,
                                      "end": 5274
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 5291,
                                        "end": 5302
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5291,
                                      "end": 5302
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 5319,
                                        "end": 5328
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5319,
                                      "end": 5328
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 5345,
                                        "end": 5354
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5345,
                                      "end": 5354
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 5371,
                                        "end": 5378
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5371,
                                      "end": 5378
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 5395,
                                        "end": 5407
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5395,
                                      "end": 5407
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 5424,
                                        "end": 5432
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5424,
                                      "end": 5432
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 5449,
                                        "end": 5463
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5486,
                                              "end": 5488
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5486,
                                            "end": 5488
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5509,
                                              "end": 5519
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5509,
                                            "end": 5519
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 5540,
                                              "end": 5550
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5540,
                                            "end": 5550
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 5571,
                                              "end": 5578
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5571,
                                            "end": 5578
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 5599,
                                              "end": 5610
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5599,
                                            "end": 5610
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5464,
                                        "end": 5628
                                      }
                                    },
                                    "loc": {
                                      "start": 5449,
                                      "end": 5628
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5217,
                                  "end": 5642
                                }
                              },
                              "loc": {
                                "start": 5213,
                                "end": 5642
                              }
                            }
                          ],
                          "loc": {
                            "start": 5116,
                            "end": 5652
                          }
                        },
                        "loc": {
                          "start": 5103,
                          "end": 5652
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5661,
                            "end": 5673
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5688,
                                  "end": 5690
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5688,
                                "end": 5690
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5703,
                                  "end": 5711
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5703,
                                "end": 5711
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5724,
                                  "end": 5735
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5724,
                                "end": 5735
                              }
                            }
                          ],
                          "loc": {
                            "start": 5674,
                            "end": 5745
                          }
                        },
                        "loc": {
                          "start": 5661,
                          "end": 5745
                        }
                      }
                    ],
                    "loc": {
                      "start": 3713,
                      "end": 5751
                    }
                  },
                  "loc": {
                    "start": 3695,
                    "end": 5751
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 5756,
                      "end": 5770
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5756,
                    "end": 5770
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 5775,
                      "end": 5787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5775,
                    "end": 5787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5792,
                      "end": 5795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5806,
                            "end": 5815
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5806,
                          "end": 5815
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 5824,
                            "end": 5833
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5824,
                          "end": 5833
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5842,
                            "end": 5851
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5842,
                          "end": 5851
                        }
                      }
                    ],
                    "loc": {
                      "start": 5796,
                      "end": 5857
                    }
                  },
                  "loc": {
                    "start": 5792,
                    "end": 5857
                  }
                }
              ],
              "loc": {
                "start": 2989,
                "end": 5859
              }
            },
            "loc": {
              "start": 2980,
              "end": 5859
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 5860,
                "end": 5871
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectVersion",
                    "loc": {
                      "start": 5878,
                      "end": 5892
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5903,
                            "end": 5905
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5903,
                          "end": 5905
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 5914,
                            "end": 5924
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5914,
                          "end": 5924
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 5933,
                            "end": 5941
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5933,
                          "end": 5941
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5950,
                            "end": 5959
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5950,
                          "end": 5959
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 5968,
                            "end": 5980
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5968,
                          "end": 5980
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 5989,
                            "end": 6001
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5989,
                          "end": 6001
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6010,
                            "end": 6014
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6029,
                                  "end": 6031
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6029,
                                "end": 6031
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6044,
                                  "end": 6053
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6044,
                                "end": 6053
                              }
                            }
                          ],
                          "loc": {
                            "start": 6015,
                            "end": 6063
                          }
                        },
                        "loc": {
                          "start": 6010,
                          "end": 6063
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6072,
                            "end": 6084
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6099,
                                  "end": 6101
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6099,
                                "end": 6101
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6114,
                                  "end": 6122
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6114,
                                "end": 6122
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6135,
                                  "end": 6146
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6135,
                                "end": 6146
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 6159,
                                  "end": 6163
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6159,
                                "end": 6163
                              }
                            }
                          ],
                          "loc": {
                            "start": 6085,
                            "end": 6173
                          }
                        },
                        "loc": {
                          "start": 6072,
                          "end": 6173
                        }
                      }
                    ],
                    "loc": {
                      "start": 5893,
                      "end": 6179
                    }
                  },
                  "loc": {
                    "start": 5878,
                    "end": 6179
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6184,
                      "end": 6186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6184,
                    "end": 6186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6191,
                      "end": 6200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6191,
                    "end": 6200
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6205,
                      "end": 6224
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6205,
                    "end": 6224
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6229,
                      "end": 6244
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6229,
                    "end": 6244
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 6249,
                      "end": 6258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6249,
                    "end": 6258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6263,
                      "end": 6274
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6263,
                    "end": 6274
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 6279,
                      "end": 6290
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6279,
                    "end": 6290
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6295,
                      "end": 6299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6295,
                    "end": 6299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 6304,
                      "end": 6310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6304,
                    "end": 6310
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 6315,
                      "end": 6325
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6315,
                    "end": 6325
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 6330,
                      "end": 6342
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6356,
                            "end": 6372
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6353,
                          "end": 6372
                        }
                      }
                    ],
                    "loc": {
                      "start": 6343,
                      "end": 6378
                    }
                  },
                  "loc": {
                    "start": 6330,
                    "end": 6378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 6383,
                      "end": 6387
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6401,
                            "end": 6409
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6398,
                          "end": 6409
                        }
                      }
                    ],
                    "loc": {
                      "start": 6388,
                      "end": 6415
                    }
                  },
                  "loc": {
                    "start": 6383,
                    "end": 6415
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6420,
                      "end": 6423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6434,
                            "end": 6443
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6434,
                          "end": 6443
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6452,
                            "end": 6461
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6452,
                          "end": 6461
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6470,
                            "end": 6477
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6470,
                          "end": 6477
                        }
                      }
                    ],
                    "loc": {
                      "start": 6424,
                      "end": 6483
                    }
                  },
                  "loc": {
                    "start": 6420,
                    "end": 6483
                  }
                }
              ],
              "loc": {
                "start": 5872,
                "end": 6485
              }
            },
            "loc": {
              "start": 5860,
              "end": 6485
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 6486,
                "end": 6497
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineVersion",
                    "loc": {
                      "start": 6504,
                      "end": 6518
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6529,
                            "end": 6531
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6529,
                          "end": 6531
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 6540,
                            "end": 6550
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6540,
                          "end": 6550
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 6559,
                            "end": 6572
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6559,
                          "end": 6572
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 6581,
                            "end": 6591
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6581,
                          "end": 6591
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 6600,
                            "end": 6609
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6600,
                          "end": 6609
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 6618,
                            "end": 6626
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6618,
                          "end": 6626
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6635,
                            "end": 6644
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6635,
                          "end": 6644
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6653,
                            "end": 6657
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6672,
                                  "end": 6674
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6672,
                                "end": 6674
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 6687,
                                  "end": 6697
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6687,
                                "end": 6697
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6710,
                                  "end": 6719
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6710,
                                "end": 6719
                              }
                            }
                          ],
                          "loc": {
                            "start": 6658,
                            "end": 6729
                          }
                        },
                        "loc": {
                          "start": 6653,
                          "end": 6729
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6738,
                            "end": 6750
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6765,
                                  "end": 6767
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6765,
                                "end": 6767
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6780,
                                  "end": 6788
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6780,
                                "end": 6788
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6801,
                                  "end": 6812
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6801,
                                "end": 6812
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 6825,
                                  "end": 6837
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6825,
                                "end": 6837
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 6850,
                                  "end": 6854
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6850,
                                "end": 6854
                              }
                            }
                          ],
                          "loc": {
                            "start": 6751,
                            "end": 6864
                          }
                        },
                        "loc": {
                          "start": 6738,
                          "end": 6864
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6873,
                            "end": 6885
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6873,
                          "end": 6885
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6894,
                            "end": 6906
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6894,
                          "end": 6906
                        }
                      }
                    ],
                    "loc": {
                      "start": 6519,
                      "end": 6912
                    }
                  },
                  "loc": {
                    "start": 6504,
                    "end": 6912
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6917,
                      "end": 6919
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6917,
                    "end": 6919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6924,
                      "end": 6933
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6924,
                    "end": 6933
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6938,
                      "end": 6957
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6938,
                    "end": 6957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6962,
                      "end": 6977
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6962,
                    "end": 6977
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 6982,
                      "end": 6991
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6982,
                    "end": 6991
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6996,
                      "end": 7007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6996,
                    "end": 7007
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7012,
                      "end": 7023
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7012,
                    "end": 7023
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7028,
                      "end": 7032
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7028,
                    "end": 7032
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7037,
                      "end": 7043
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7037,
                    "end": 7043
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7048,
                      "end": 7058
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7048,
                    "end": 7058
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 7063,
                      "end": 7074
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7063,
                    "end": 7074
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 7079,
                      "end": 7098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7079,
                    "end": 7098
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 7103,
                      "end": 7115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7129,
                            "end": 7145
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7126,
                          "end": 7145
                        }
                      }
                    ],
                    "loc": {
                      "start": 7116,
                      "end": 7151
                    }
                  },
                  "loc": {
                    "start": 7103,
                    "end": 7151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 7156,
                      "end": 7160
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 7174,
                            "end": 7182
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7171,
                          "end": 7182
                        }
                      }
                    ],
                    "loc": {
                      "start": 7161,
                      "end": 7188
                    }
                  },
                  "loc": {
                    "start": 7156,
                    "end": 7188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7193,
                      "end": 7196
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 7207,
                            "end": 7216
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7207,
                          "end": 7216
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7225,
                            "end": 7234
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7225,
                          "end": 7234
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7243,
                            "end": 7250
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7243,
                          "end": 7250
                        }
                      }
                    ],
                    "loc": {
                      "start": 7197,
                      "end": 7256
                    }
                  },
                  "loc": {
                    "start": 7193,
                    "end": 7256
                  }
                }
              ],
              "loc": {
                "start": 6498,
                "end": 7258
              }
            },
            "loc": {
              "start": 6486,
              "end": 7258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7259,
                "end": 7261
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7259,
              "end": 7261
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7262,
                "end": 7272
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7262,
              "end": 7272
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7273,
                "end": 7283
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7273,
              "end": 7283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 7284,
                "end": 7293
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7284,
              "end": 7293
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 7294,
                "end": 7301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7294,
              "end": 7301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 7302,
                "end": 7310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7302,
              "end": 7310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 7311,
                "end": 7321
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7328,
                      "end": 7330
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7328,
                    "end": 7330
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 7335,
                      "end": 7352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7335,
                    "end": 7352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 7357,
                      "end": 7369
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7357,
                    "end": 7369
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 7374,
                      "end": 7384
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7374,
                    "end": 7384
                  }
                }
              ],
              "loc": {
                "start": 7322,
                "end": 7386
              }
            },
            "loc": {
              "start": 7311,
              "end": 7386
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 7387,
                "end": 7398
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7405,
                      "end": 7407
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7405,
                    "end": 7407
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 7412,
                      "end": 7426
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7412,
                    "end": 7426
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 7431,
                      "end": 7439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7431,
                    "end": 7439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 7444,
                      "end": 7453
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7444,
                    "end": 7453
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 7458,
                      "end": 7468
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7458,
                    "end": 7468
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 7473,
                      "end": 7478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7473,
                    "end": 7478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 7483,
                      "end": 7490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7483,
                    "end": 7490
                  }
                }
              ],
              "loc": {
                "start": 7399,
                "end": 7492
              }
            },
            "loc": {
              "start": 7387,
              "end": 7492
            }
          }
        ],
        "loc": {
          "start": 1941,
          "end": 7494
        }
      },
      "loc": {
        "start": 1906,
        "end": 7494
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7504,
          "end": 7512
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7516,
            "end": 7520
          }
        },
        "loc": {
          "start": 7516,
          "end": 7520
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7523,
                "end": 7525
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7523,
              "end": 7525
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7526,
                "end": 7536
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7526,
              "end": 7536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7537,
                "end": 7547
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7537,
              "end": 7547
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7548,
                "end": 7559
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7548,
              "end": 7559
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7560,
                "end": 7566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7560,
              "end": 7566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7567,
                "end": 7572
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7567,
              "end": 7572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7573,
                "end": 7593
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7573,
              "end": 7593
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7594,
                "end": 7598
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7594,
              "end": 7598
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7599,
                "end": 7611
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7599,
              "end": 7611
            }
          }
        ],
        "loc": {
          "start": 7521,
          "end": 7613
        }
      },
      "loc": {
        "start": 7495,
        "end": 7613
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "home",
      "loc": {
        "start": 7621,
        "end": 7625
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7627,
              "end": 7632
            }
          },
          "loc": {
            "start": 7626,
            "end": 7632
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "HomeInput",
              "loc": {
                "start": 7634,
                "end": 7643
              }
            },
            "loc": {
              "start": 7634,
              "end": 7643
            }
          },
          "loc": {
            "start": 7634,
            "end": 7644
          }
        },
        "directives": [],
        "loc": {
          "start": 7626,
          "end": 7644
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "home",
            "loc": {
              "start": 7650,
              "end": 7654
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7655,
                  "end": 7660
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7663,
                    "end": 7668
                  }
                },
                "loc": {
                  "start": 7662,
                  "end": 7668
                }
              },
              "loc": {
                "start": 7655,
                "end": 7668
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "recommended",
                  "loc": {
                    "start": 7676,
                    "end": 7687
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Resource_list",
                        "loc": {
                          "start": 7701,
                          "end": 7714
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7698,
                        "end": 7714
                      }
                    }
                  ],
                  "loc": {
                    "start": 7688,
                    "end": 7720
                  }
                },
                "loc": {
                  "start": 7676,
                  "end": 7720
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 7725,
                    "end": 7734
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Reminder_full",
                        "loc": {
                          "start": 7748,
                          "end": 7761
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7745,
                        "end": 7761
                      }
                    }
                  ],
                  "loc": {
                    "start": 7735,
                    "end": 7767
                  }
                },
                "loc": {
                  "start": 7725,
                  "end": 7767
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 7772,
                    "end": 7781
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Resource_list",
                        "loc": {
                          "start": 7795,
                          "end": 7808
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7792,
                        "end": 7808
                      }
                    }
                  ],
                  "loc": {
                    "start": 7782,
                    "end": 7814
                  }
                },
                "loc": {
                  "start": 7772,
                  "end": 7814
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 7819,
                    "end": 7828
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Schedule_list",
                        "loc": {
                          "start": 7842,
                          "end": 7855
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7839,
                        "end": 7855
                      }
                    }
                  ],
                  "loc": {
                    "start": 7829,
                    "end": 7861
                  }
                },
                "loc": {
                  "start": 7819,
                  "end": 7861
                }
              }
            ],
            "loc": {
              "start": 7670,
              "end": 7865
            }
          },
          "loc": {
            "start": 7650,
            "end": 7865
          }
        }
      ],
      "loc": {
        "start": 7646,
        "end": 7867
      }
    },
    "loc": {
      "start": 7615,
      "end": 7867
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
