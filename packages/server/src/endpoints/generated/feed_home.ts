export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 7806,
          "end": 7810
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7811,
              "end": 7816
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7819,
                "end": 7824
              }
            },
            "loc": {
              "start": 7818,
              "end": 7824
            }
          },
          "loc": {
            "start": 7811,
            "end": 7824
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
                "start": 7832,
                "end": 7843
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
                      "start": 7857,
                      "end": 7870
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7854,
                    "end": 7870
                  }
                }
              ],
              "loc": {
                "start": 7844,
                "end": 7876
              }
            },
            "loc": {
              "start": 7832,
              "end": 7876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 7881,
                "end": 7890
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
                      "start": 7904,
                      "end": 7917
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7901,
                    "end": 7917
                  }
                }
              ],
              "loc": {
                "start": 7891,
                "end": 7923
              }
            },
            "loc": {
              "start": 7881,
              "end": 7923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 7928,
                "end": 7937
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
                      "start": 7951,
                      "end": 7964
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7948,
                    "end": 7964
                  }
                }
              ],
              "loc": {
                "start": 7938,
                "end": 7970
              }
            },
            "loc": {
              "start": 7928,
              "end": 7970
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 7975,
                "end": 7984
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
                      "start": 7998,
                      "end": 8011
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7995,
                    "end": 8011
                  }
                }
              ],
              "loc": {
                "start": 7985,
                "end": 8017
              }
            },
            "loc": {
              "start": 7975,
              "end": 8017
            }
          }
        ],
        "loc": {
          "start": 7826,
          "end": 8021
        }
      },
      "loc": {
        "start": 7806,
        "end": 8021
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
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1506,
                      "end": 1509
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
                            "start": 1524,
                            "end": 1533
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1524,
                          "end": 1533
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1546,
                            "end": 1553
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1546,
                          "end": 1553
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1566,
                            "end": 1575
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1566,
                          "end": 1575
                        }
                      }
                    ],
                    "loc": {
                      "start": 1510,
                      "end": 1585
                    }
                  },
                  "loc": {
                    "start": 1506,
                    "end": 1585
                  }
                }
              ],
              "loc": {
                "start": 827,
                "end": 1591
              }
            },
            "loc": {
              "start": 817,
              "end": 1591
            }
          }
        ],
        "loc": {
          "start": 774,
          "end": 1593
        }
      },
      "loc": {
        "start": 761,
        "end": 1593
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1633,
          "end": 1635
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1633,
        "end": 1635
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1636,
          "end": 1641
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1636,
        "end": 1641
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 1642,
          "end": 1646
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1642,
        "end": 1646
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 1647,
          "end": 1654
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1647,
        "end": 1654
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1655,
          "end": 1667
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
                "start": 1674,
                "end": 1676
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1674,
              "end": 1676
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1681,
                "end": 1689
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1681,
              "end": 1689
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1694,
                "end": 1705
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1694,
              "end": 1705
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1710,
                "end": 1714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1710,
              "end": 1714
            }
          }
        ],
        "loc": {
          "start": 1668,
          "end": 1716
        }
      },
      "loc": {
        "start": 1655,
        "end": 1716
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1758,
          "end": 1760
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1758,
        "end": 1760
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1761,
          "end": 1771
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1761,
        "end": 1771
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1772,
          "end": 1782
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1772,
        "end": 1782
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 1783,
          "end": 1792
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1783,
        "end": 1792
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 1793,
          "end": 1800
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1793,
        "end": 1800
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 1801,
          "end": 1809
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1801,
        "end": 1809
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 1810,
          "end": 1820
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
                "start": 1827,
                "end": 1829
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1827,
              "end": 1829
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 1834,
                "end": 1851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1834,
              "end": 1851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 1856,
                "end": 1868
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1856,
              "end": 1868
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 1873,
                "end": 1883
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1873,
              "end": 1883
            }
          }
        ],
        "loc": {
          "start": 1821,
          "end": 1885
        }
      },
      "loc": {
        "start": 1810,
        "end": 1885
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 1886,
          "end": 1897
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
                "start": 1904,
                "end": 1906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1904,
              "end": 1906
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 1911,
                "end": 1925
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1911,
              "end": 1925
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 1930,
                "end": 1938
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1930,
              "end": 1938
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 1943,
                "end": 1952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1943,
              "end": 1952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 1957,
                "end": 1967
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1957,
              "end": 1967
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 1972,
                "end": 1977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1972,
              "end": 1977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 1982,
                "end": 1989
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1982,
              "end": 1989
            }
          }
        ],
        "loc": {
          "start": 1898,
          "end": 1991
        }
      },
      "loc": {
        "start": 1886,
        "end": 1991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2031,
          "end": 2037
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
                "start": 2047,
                "end": 2057
              }
            },
            "directives": [],
            "loc": {
              "start": 2044,
              "end": 2057
            }
          }
        ],
        "loc": {
          "start": 2038,
          "end": 2059
        }
      },
      "loc": {
        "start": 2031,
        "end": 2059
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModes",
        "loc": {
          "start": 2060,
          "end": 2070
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
                "start": 2077,
                "end": 2083
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
                      "start": 2094,
                      "end": 2096
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2094,
                    "end": 2096
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "color",
                    "loc": {
                      "start": 2105,
                      "end": 2110
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2105,
                    "end": 2110
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 2119,
                      "end": 2124
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2119,
                    "end": 2124
                  }
                }
              ],
              "loc": {
                "start": 2084,
                "end": 2130
              }
            },
            "loc": {
              "start": 2077,
              "end": 2130
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 2135,
                "end": 2147
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
                      "start": 2158,
                      "end": 2160
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2158,
                    "end": 2160
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2169,
                      "end": 2179
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2169,
                    "end": 2179
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2188,
                      "end": 2198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2188,
                    "end": 2198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminders",
                    "loc": {
                      "start": 2207,
                      "end": 2216
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
                            "start": 2231,
                            "end": 2233
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2231,
                          "end": 2233
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2246,
                            "end": 2256
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2246,
                          "end": 2256
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2269,
                            "end": 2279
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2269,
                          "end": 2279
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2292,
                            "end": 2296
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2292,
                          "end": 2296
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2309,
                            "end": 2320
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2309,
                          "end": 2320
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dueDate",
                          "loc": {
                            "start": 2333,
                            "end": 2340
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2333,
                          "end": 2340
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 2353,
                            "end": 2358
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2353,
                          "end": 2358
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 2371,
                            "end": 2381
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2371,
                          "end": 2381
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderItems",
                          "loc": {
                            "start": 2394,
                            "end": 2407
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
                                  "start": 2426,
                                  "end": 2428
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2426,
                                "end": 2428
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2445,
                                  "end": 2455
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2445,
                                "end": 2455
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2472,
                                  "end": 2482
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2472,
                                "end": 2482
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2499,
                                  "end": 2503
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2499,
                                "end": 2503
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2520,
                                  "end": 2531
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2520,
                                "end": 2531
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 2548,
                                  "end": 2555
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2548,
                                "end": 2555
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2572,
                                  "end": 2577
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2572,
                                "end": 2577
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 2594,
                                  "end": 2604
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2594,
                                "end": 2604
                              }
                            }
                          ],
                          "loc": {
                            "start": 2408,
                            "end": 2618
                          }
                        },
                        "loc": {
                          "start": 2394,
                          "end": 2618
                        }
                      }
                    ],
                    "loc": {
                      "start": 2217,
                      "end": 2628
                    }
                  },
                  "loc": {
                    "start": 2207,
                    "end": 2628
                  }
                }
              ],
              "loc": {
                "start": 2148,
                "end": 2634
              }
            },
            "loc": {
              "start": 2135,
              "end": 2634
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resourceList",
              "loc": {
                "start": 2639,
                "end": 2651
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
                      "start": 2662,
                      "end": 2664
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2662,
                    "end": 2664
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2673,
                      "end": 2683
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2673,
                    "end": 2683
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2692,
                      "end": 2704
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
                            "start": 2719,
                            "end": 2721
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2719,
                          "end": 2721
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2734,
                            "end": 2742
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2734,
                          "end": 2742
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2755,
                            "end": 2766
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2755,
                          "end": 2766
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2779,
                            "end": 2783
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2779,
                          "end": 2783
                        }
                      }
                    ],
                    "loc": {
                      "start": 2705,
                      "end": 2793
                    }
                  },
                  "loc": {
                    "start": 2692,
                    "end": 2793
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resources",
                    "loc": {
                      "start": 2802,
                      "end": 2811
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
                            "start": 2826,
                            "end": 2828
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2826,
                          "end": 2828
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 2841,
                            "end": 2846
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2841,
                          "end": 2846
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 2859,
                            "end": 2863
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2859,
                          "end": 2863
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "usedFor",
                          "loc": {
                            "start": 2876,
                            "end": 2883
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2876,
                          "end": 2883
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 2896,
                            "end": 2908
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
                                  "start": 2927,
                                  "end": 2929
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2927,
                                "end": 2929
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 2946,
                                  "end": 2954
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2946,
                                "end": 2954
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2971,
                                  "end": 2982
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2971,
                                "end": 2982
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2999,
                                  "end": 3003
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2999,
                                "end": 3003
                              }
                            }
                          ],
                          "loc": {
                            "start": 2909,
                            "end": 3017
                          }
                        },
                        "loc": {
                          "start": 2896,
                          "end": 3017
                        }
                      }
                    ],
                    "loc": {
                      "start": 2812,
                      "end": 3027
                    }
                  },
                  "loc": {
                    "start": 2802,
                    "end": 3027
                  }
                }
              ],
              "loc": {
                "start": 2652,
                "end": 3033
              }
            },
            "loc": {
              "start": 2639,
              "end": 3033
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3038,
                "end": 3040
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3038,
              "end": 3040
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3045,
                "end": 3049
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3045,
              "end": 3049
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3054,
                "end": 3065
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3054,
              "end": 3065
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3070,
                "end": 3073
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
                      "start": 3084,
                      "end": 3093
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3084,
                    "end": 3093
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3102,
                      "end": 3109
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3102,
                    "end": 3109
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3118,
                      "end": 3127
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3118,
                    "end": 3127
                  }
                }
              ],
              "loc": {
                "start": 3074,
                "end": 3133
              }
            },
            "loc": {
              "start": 3070,
              "end": 3133
            }
          }
        ],
        "loc": {
          "start": 2071,
          "end": 3135
        }
      },
      "loc": {
        "start": 2060,
        "end": 3135
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetings",
        "loc": {
          "start": 3136,
          "end": 3144
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
                "start": 3151,
                "end": 3157
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
                      "start": 3171,
                      "end": 3181
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3168,
                    "end": 3181
                  }
                }
              ],
              "loc": {
                "start": 3158,
                "end": 3187
              }
            },
            "loc": {
              "start": 3151,
              "end": 3187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3192,
                "end": 3204
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
                      "start": 3215,
                      "end": 3217
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3215,
                    "end": 3217
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3226,
                      "end": 3234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3226,
                    "end": 3234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3243,
                      "end": 3254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3243,
                    "end": 3254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "link",
                    "loc": {
                      "start": 3263,
                      "end": 3267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3263,
                    "end": 3267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3276,
                      "end": 3280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3276,
                    "end": 3280
                  }
                }
              ],
              "loc": {
                "start": 3205,
                "end": 3286
              }
            },
            "loc": {
              "start": 3192,
              "end": 3286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3291,
                "end": 3293
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3291,
              "end": 3293
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3298,
                "end": 3308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3298,
              "end": 3308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3313,
                "end": 3323
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3313,
              "end": 3323
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 3328,
                "end": 3350
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3328,
              "end": 3350
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnOrganizationProfile",
              "loc": {
                "start": 3355,
                "end": 3380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3355,
              "end": 3380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 3385,
                "end": 3397
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
                      "start": 3408,
                      "end": 3410
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3408,
                    "end": 3410
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3419,
                      "end": 3430
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3419,
                    "end": 3430
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3439,
                      "end": 3445
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3439,
                    "end": 3445
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3454,
                      "end": 3466
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3454,
                    "end": 3466
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 3475,
                      "end": 3478
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
                            "start": 3493,
                            "end": 3506
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3493,
                          "end": 3506
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 3519,
                            "end": 3528
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3519,
                          "end": 3528
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 3541,
                            "end": 3552
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3541,
                          "end": 3552
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 3565,
                            "end": 3574
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3565,
                          "end": 3574
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 3587,
                            "end": 3596
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3587,
                          "end": 3596
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 3609,
                            "end": 3616
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3609,
                          "end": 3616
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 3629,
                            "end": 3641
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3629,
                          "end": 3641
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 3654,
                            "end": 3662
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3654,
                          "end": 3662
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 3675,
                            "end": 3689
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
                                  "start": 3708,
                                  "end": 3710
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3708,
                                "end": 3710
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3727,
                                  "end": 3737
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3727,
                                "end": 3737
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3754,
                                  "end": 3764
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3754,
                                "end": 3764
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3781,
                                  "end": 3788
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3781,
                                "end": 3788
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3805,
                                  "end": 3816
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3805,
                                "end": 3816
                              }
                            }
                          ],
                          "loc": {
                            "start": 3690,
                            "end": 3830
                          }
                        },
                        "loc": {
                          "start": 3675,
                          "end": 3830
                        }
                      }
                    ],
                    "loc": {
                      "start": 3479,
                      "end": 3840
                    }
                  },
                  "loc": {
                    "start": 3475,
                    "end": 3840
                  }
                }
              ],
              "loc": {
                "start": 3398,
                "end": 3846
              }
            },
            "loc": {
              "start": 3385,
              "end": 3846
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 3851,
                "end": 3868
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
                      "start": 3879,
                      "end": 3886
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
                            "start": 3901,
                            "end": 3903
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3901,
                          "end": 3903
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3916,
                            "end": 3926
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3916,
                          "end": 3926
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3939,
                            "end": 3949
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3939,
                          "end": 3949
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3962,
                            "end": 3969
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3962,
                          "end": 3969
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3982,
                            "end": 3993
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3982,
                          "end": 3993
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 4006,
                            "end": 4011
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
                                  "start": 4030,
                                  "end": 4032
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4030,
                                "end": 4032
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4049,
                                  "end": 4059
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4049,
                                "end": 4059
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4076,
                                  "end": 4086
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4076,
                                "end": 4086
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4103,
                                  "end": 4107
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4103,
                                "end": 4107
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4124,
                                  "end": 4135
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4124,
                                "end": 4135
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 4152,
                                  "end": 4164
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4152,
                                "end": 4164
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "organization",
                                "loc": {
                                  "start": 4181,
                                  "end": 4193
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
                                        "start": 4216,
                                        "end": 4218
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4216,
                                      "end": 4218
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 4239,
                                        "end": 4250
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4239,
                                      "end": 4250
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 4271,
                                        "end": 4277
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4271,
                                      "end": 4277
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 4298,
                                        "end": 4310
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4298,
                                      "end": 4310
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 4331,
                                        "end": 4334
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
                                              "start": 4361,
                                              "end": 4374
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4361,
                                            "end": 4374
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 4399,
                                              "end": 4408
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4399,
                                            "end": 4408
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 4433,
                                              "end": 4444
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4433,
                                            "end": 4444
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 4469,
                                              "end": 4478
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4469,
                                            "end": 4478
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 4503,
                                              "end": 4512
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4503,
                                            "end": 4512
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 4537,
                                              "end": 4544
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4537,
                                            "end": 4544
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 4569,
                                              "end": 4581
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4569,
                                            "end": 4581
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 4606,
                                              "end": 4614
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4606,
                                            "end": 4614
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 4639,
                                              "end": 4653
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
                                                    "start": 4684,
                                                    "end": 4686
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4684,
                                                  "end": 4686
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 4715,
                                                    "end": 4725
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4715,
                                                  "end": 4725
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 4754,
                                                    "end": 4764
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4754,
                                                  "end": 4764
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 4793,
                                                    "end": 4800
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4793,
                                                  "end": 4800
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 4829,
                                                    "end": 4840
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4829,
                                                  "end": 4840
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4654,
                                              "end": 4866
                                            }
                                          },
                                          "loc": {
                                            "start": 4639,
                                            "end": 4866
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4335,
                                        "end": 4888
                                      }
                                    },
                                    "loc": {
                                      "start": 4331,
                                      "end": 4888
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4194,
                                  "end": 4906
                                }
                              },
                              "loc": {
                                "start": 4181,
                                "end": 4906
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4923,
                                  "end": 4935
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
                                        "start": 4958,
                                        "end": 4960
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4958,
                                      "end": 4960
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4981,
                                        "end": 4989
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4981,
                                      "end": 4989
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5010,
                                        "end": 5021
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5010,
                                      "end": 5021
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4936,
                                  "end": 5039
                                }
                              },
                              "loc": {
                                "start": 4923,
                                "end": 5039
                              }
                            }
                          ],
                          "loc": {
                            "start": 4012,
                            "end": 5053
                          }
                        },
                        "loc": {
                          "start": 4006,
                          "end": 5053
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 5066,
                            "end": 5069
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
                                  "start": 5088,
                                  "end": 5097
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5088,
                                "end": 5097
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 5114,
                                  "end": 5123
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5114,
                                "end": 5123
                              }
                            }
                          ],
                          "loc": {
                            "start": 5070,
                            "end": 5137
                          }
                        },
                        "loc": {
                          "start": 5066,
                          "end": 5137
                        }
                      }
                    ],
                    "loc": {
                      "start": 3887,
                      "end": 5147
                    }
                  },
                  "loc": {
                    "start": 3879,
                    "end": 5147
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5156,
                      "end": 5158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5156,
                    "end": 5158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5167,
                      "end": 5177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5167,
                    "end": 5177
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5186,
                      "end": 5196
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5186,
                    "end": 5196
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5205,
                      "end": 5209
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5205,
                    "end": 5209
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 5218,
                      "end": 5229
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5218,
                    "end": 5229
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 5238,
                      "end": 5250
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5238,
                    "end": 5250
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 5259,
                      "end": 5271
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
                            "start": 5286,
                            "end": 5288
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5286,
                          "end": 5288
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 5301,
                            "end": 5312
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5301,
                          "end": 5312
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 5325,
                            "end": 5331
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5325,
                          "end": 5331
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 5344,
                            "end": 5356
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5344,
                          "end": 5356
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 5369,
                            "end": 5372
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
                                  "start": 5391,
                                  "end": 5404
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5391,
                                "end": 5404
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 5421,
                                  "end": 5430
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5421,
                                "end": 5430
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 5447,
                                  "end": 5458
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5447,
                                "end": 5458
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 5475,
                                  "end": 5484
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5475,
                                "end": 5484
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 5501,
                                  "end": 5510
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5501,
                                "end": 5510
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 5527,
                                  "end": 5534
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5527,
                                "end": 5534
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 5551,
                                  "end": 5563
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5551,
                                "end": 5563
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 5580,
                                  "end": 5588
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5580,
                                "end": 5588
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 5605,
                                  "end": 5619
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
                                        "start": 5642,
                                        "end": 5644
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5642,
                                      "end": 5644
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5665,
                                        "end": 5675
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5665,
                                      "end": 5675
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 5696,
                                        "end": 5706
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5696,
                                      "end": 5706
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 5727,
                                        "end": 5734
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5727,
                                      "end": 5734
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 5755,
                                        "end": 5766
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5755,
                                      "end": 5766
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5620,
                                  "end": 5784
                                }
                              },
                              "loc": {
                                "start": 5605,
                                "end": 5784
                              }
                            }
                          ],
                          "loc": {
                            "start": 5373,
                            "end": 5798
                          }
                        },
                        "loc": {
                          "start": 5369,
                          "end": 5798
                        }
                      }
                    ],
                    "loc": {
                      "start": 5272,
                      "end": 5808
                    }
                  },
                  "loc": {
                    "start": 5259,
                    "end": 5808
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5817,
                      "end": 5829
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
                            "start": 5844,
                            "end": 5846
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5844,
                          "end": 5846
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5859,
                            "end": 5867
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5859,
                          "end": 5867
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5880,
                            "end": 5891
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5880,
                          "end": 5891
                        }
                      }
                    ],
                    "loc": {
                      "start": 5830,
                      "end": 5901
                    }
                  },
                  "loc": {
                    "start": 5817,
                    "end": 5901
                  }
                }
              ],
              "loc": {
                "start": 3869,
                "end": 5907
              }
            },
            "loc": {
              "start": 3851,
              "end": 5907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 5912,
                "end": 5926
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5912,
              "end": 5926
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 5931,
                "end": 5943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5931,
              "end": 5943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5948,
                "end": 5951
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
                      "start": 5962,
                      "end": 5971
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5962,
                    "end": 5971
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 5980,
                      "end": 5989
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5980,
                    "end": 5989
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5998,
                      "end": 6007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5998,
                    "end": 6007
                  }
                }
              ],
              "loc": {
                "start": 5952,
                "end": 6013
              }
            },
            "loc": {
              "start": 5948,
              "end": 6013
            }
          }
        ],
        "loc": {
          "start": 3145,
          "end": 6015
        }
      },
      "loc": {
        "start": 3136,
        "end": 6015
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 6016,
          "end": 6027
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
                "start": 6034,
                "end": 6048
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
                      "start": 6059,
                      "end": 6061
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6059,
                    "end": 6061
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 6070,
                      "end": 6080
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6070,
                    "end": 6080
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6089,
                      "end": 6097
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6089,
                    "end": 6097
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6106,
                      "end": 6115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6106,
                    "end": 6115
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6124,
                      "end": 6136
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6124,
                    "end": 6136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6145,
                      "end": 6157
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6145,
                    "end": 6157
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6166,
                      "end": 6170
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
                            "start": 6185,
                            "end": 6187
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6185,
                          "end": 6187
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6200,
                            "end": 6209
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6200,
                          "end": 6209
                        }
                      }
                    ],
                    "loc": {
                      "start": 6171,
                      "end": 6219
                    }
                  },
                  "loc": {
                    "start": 6166,
                    "end": 6219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6228,
                      "end": 6240
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
                            "start": 6255,
                            "end": 6257
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6255,
                          "end": 6257
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6270,
                            "end": 6278
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6270,
                          "end": 6278
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6291,
                            "end": 6302
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6291,
                          "end": 6302
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6315,
                            "end": 6319
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6315,
                          "end": 6319
                        }
                      }
                    ],
                    "loc": {
                      "start": 6241,
                      "end": 6329
                    }
                  },
                  "loc": {
                    "start": 6228,
                    "end": 6329
                  }
                }
              ],
              "loc": {
                "start": 6049,
                "end": 6335
              }
            },
            "loc": {
              "start": 6034,
              "end": 6335
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6340,
                "end": 6342
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6340,
              "end": 6342
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6347,
                "end": 6356
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6347,
              "end": 6356
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6361,
                "end": 6380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6361,
              "end": 6380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6385,
                "end": 6400
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6385,
              "end": 6400
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 6405,
                "end": 6414
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6405,
              "end": 6414
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 6419,
                "end": 6430
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6419,
              "end": 6430
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 6435,
                "end": 6446
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6435,
              "end": 6446
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6451,
                "end": 6455
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6451,
              "end": 6455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 6460,
                "end": 6466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6460,
              "end": 6466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 6471,
                "end": 6481
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6471,
              "end": 6481
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 6486,
                "end": 6498
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
                      "start": 6512,
                      "end": 6528
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6509,
                    "end": 6528
                  }
                }
              ],
              "loc": {
                "start": 6499,
                "end": 6534
              }
            },
            "loc": {
              "start": 6486,
              "end": 6534
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 6539,
                "end": 6543
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
                      "start": 6557,
                      "end": 6565
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6554,
                    "end": 6565
                  }
                }
              ],
              "loc": {
                "start": 6544,
                "end": 6571
              }
            },
            "loc": {
              "start": 6539,
              "end": 6571
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6576,
                "end": 6579
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
                      "start": 6590,
                      "end": 6599
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6590,
                    "end": 6599
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6608,
                      "end": 6617
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6608,
                    "end": 6617
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6626,
                      "end": 6633
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6626,
                    "end": 6633
                  }
                }
              ],
              "loc": {
                "start": 6580,
                "end": 6639
              }
            },
            "loc": {
              "start": 6576,
              "end": 6639
            }
          }
        ],
        "loc": {
          "start": 6028,
          "end": 6641
        }
      },
      "loc": {
        "start": 6016,
        "end": 6641
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 6642,
          "end": 6653
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
                "start": 6660,
                "end": 6674
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
                      "start": 6685,
                      "end": 6687
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6685,
                    "end": 6687
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 6696,
                      "end": 6706
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6696,
                    "end": 6706
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 6715,
                      "end": 6728
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6715,
                    "end": 6728
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6737,
                      "end": 6747
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6737,
                    "end": 6747
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 6756,
                      "end": 6765
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6756,
                    "end": 6765
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6774,
                      "end": 6782
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6774,
                    "end": 6782
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6791,
                      "end": 6800
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6791,
                    "end": 6800
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6809,
                      "end": 6813
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
                            "start": 6828,
                            "end": 6830
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6828,
                          "end": 6830
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 6843,
                            "end": 6853
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6843,
                          "end": 6853
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6866,
                            "end": 6875
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6866,
                          "end": 6875
                        }
                      }
                    ],
                    "loc": {
                      "start": 6814,
                      "end": 6885
                    }
                  },
                  "loc": {
                    "start": 6809,
                    "end": 6885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6894,
                      "end": 6906
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
                            "start": 6921,
                            "end": 6923
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6921,
                          "end": 6923
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6936,
                            "end": 6944
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6936,
                          "end": 6944
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6957,
                            "end": 6968
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6957,
                          "end": 6968
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 6981,
                            "end": 6993
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6981,
                          "end": 6993
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7006,
                            "end": 7010
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7006,
                          "end": 7010
                        }
                      }
                    ],
                    "loc": {
                      "start": 6907,
                      "end": 7020
                    }
                  },
                  "loc": {
                    "start": 6894,
                    "end": 7020
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 7029,
                      "end": 7041
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7029,
                    "end": 7041
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 7050,
                      "end": 7062
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7050,
                    "end": 7062
                  }
                }
              ],
              "loc": {
                "start": 6675,
                "end": 7068
              }
            },
            "loc": {
              "start": 6660,
              "end": 7068
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7073,
                "end": 7075
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7073,
              "end": 7075
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7080,
                "end": 7089
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7080,
              "end": 7089
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 7094,
                "end": 7113
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7094,
              "end": 7113
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 7118,
                "end": 7133
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7118,
              "end": 7133
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 7138,
                "end": 7147
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7138,
              "end": 7147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 7152,
                "end": 7163
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7152,
              "end": 7163
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7168,
                "end": 7179
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7168,
              "end": 7179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7184,
                "end": 7188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7184,
              "end": 7188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7193,
                "end": 7199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7193,
              "end": 7199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7204,
                "end": 7214
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7204,
              "end": 7214
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 7219,
                "end": 7230
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7219,
              "end": 7230
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 7235,
                "end": 7254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7235,
              "end": 7254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 7259,
                "end": 7271
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
                      "start": 7285,
                      "end": 7301
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7282,
                    "end": 7301
                  }
                }
              ],
              "loc": {
                "start": 7272,
                "end": 7307
              }
            },
            "loc": {
              "start": 7259,
              "end": 7307
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 7312,
                "end": 7316
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
                      "start": 7330,
                      "end": 7338
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7327,
                    "end": 7338
                  }
                }
              ],
              "loc": {
                "start": 7317,
                "end": 7344
              }
            },
            "loc": {
              "start": 7312,
              "end": 7344
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7349,
                "end": 7352
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
                      "start": 7363,
                      "end": 7372
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7363,
                    "end": 7372
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7381,
                      "end": 7390
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7381,
                    "end": 7390
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7399,
                      "end": 7406
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7399,
                    "end": 7406
                  }
                }
              ],
              "loc": {
                "start": 7353,
                "end": 7412
              }
            },
            "loc": {
              "start": 7349,
              "end": 7412
            }
          }
        ],
        "loc": {
          "start": 6654,
          "end": 7414
        }
      },
      "loc": {
        "start": 6642,
        "end": 7414
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7415,
          "end": 7417
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7415,
        "end": 7417
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7418,
          "end": 7428
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7418,
        "end": 7428
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7429,
          "end": 7439
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7429,
        "end": 7439
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 7440,
          "end": 7449
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7440,
        "end": 7449
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 7450,
          "end": 7457
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7450,
        "end": 7457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 7458,
          "end": 7466
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7458,
        "end": 7466
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 7467,
          "end": 7477
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
                "start": 7484,
                "end": 7486
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7484,
              "end": 7486
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 7491,
                "end": 7508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7491,
              "end": 7508
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 7513,
                "end": 7525
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7513,
              "end": 7525
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 7530,
                "end": 7540
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7530,
              "end": 7540
            }
          }
        ],
        "loc": {
          "start": 7478,
          "end": 7542
        }
      },
      "loc": {
        "start": 7467,
        "end": 7542
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 7543,
          "end": 7554
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
                "start": 7561,
                "end": 7563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7561,
              "end": 7563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 7568,
                "end": 7582
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7568,
              "end": 7582
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 7587,
                "end": 7595
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7587,
              "end": 7595
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 7600,
                "end": 7609
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7600,
              "end": 7609
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 7614,
                "end": 7624
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7614,
              "end": 7624
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 7629,
                "end": 7634
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7629,
              "end": 7634
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 7639,
                "end": 7646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7639,
              "end": 7646
            }
          }
        ],
        "loc": {
          "start": 7555,
          "end": 7648
        }
      },
      "loc": {
        "start": 7543,
        "end": 7648
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7679,
          "end": 7681
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7679,
        "end": 7681
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7682,
          "end": 7692
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7682,
        "end": 7692
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7693,
          "end": 7703
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7693,
        "end": 7703
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7704,
          "end": 7715
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7704,
        "end": 7715
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7716,
          "end": 7722
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7716,
        "end": 7722
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7723,
          "end": 7728
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7723,
        "end": 7728
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7729,
          "end": 7749
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7729,
        "end": 7749
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7750,
          "end": 7754
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7750,
        "end": 7754
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7755,
          "end": 7767
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7755,
        "end": 7767
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
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 1506,
                            "end": 1509
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
                                  "start": 1524,
                                  "end": 1533
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1524,
                                "end": 1533
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 1546,
                                  "end": 1553
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1546,
                                "end": 1553
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 1566,
                                  "end": 1575
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1566,
                                "end": 1575
                              }
                            }
                          ],
                          "loc": {
                            "start": 1510,
                            "end": 1585
                          }
                        },
                        "loc": {
                          "start": 1506,
                          "end": 1585
                        }
                      }
                    ],
                    "loc": {
                      "start": 827,
                      "end": 1591
                    }
                  },
                  "loc": {
                    "start": 817,
                    "end": 1591
                  }
                }
              ],
              "loc": {
                "start": 774,
                "end": 1593
              }
            },
            "loc": {
              "start": 761,
              "end": 1593
            }
          }
        ],
        "loc": {
          "start": 575,
          "end": 1595
        }
      },
      "loc": {
        "start": 540,
        "end": 1595
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1605,
          "end": 1618
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1622,
            "end": 1630
          }
        },
        "loc": {
          "start": 1622,
          "end": 1630
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
                "start": 1633,
                "end": 1635
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1633,
              "end": 1635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1636,
                "end": 1641
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1636,
              "end": 1641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1642,
                "end": 1646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1642,
              "end": 1646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1647,
                "end": 1654
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1647,
              "end": 1654
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1655,
                "end": 1667
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
                      "start": 1674,
                      "end": 1676
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1674,
                    "end": 1676
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1681,
                      "end": 1689
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1681,
                    "end": 1689
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1694,
                      "end": 1705
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1694,
                    "end": 1705
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1710,
                      "end": 1714
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1710,
                    "end": 1714
                  }
                }
              ],
              "loc": {
                "start": 1668,
                "end": 1716
              }
            },
            "loc": {
              "start": 1655,
              "end": 1716
            }
          }
        ],
        "loc": {
          "start": 1631,
          "end": 1718
        }
      },
      "loc": {
        "start": 1596,
        "end": 1718
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 1728,
          "end": 1743
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1747,
            "end": 1755
          }
        },
        "loc": {
          "start": 1747,
          "end": 1755
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
                "start": 1758,
                "end": 1760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1758,
              "end": 1760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1761,
                "end": 1771
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1761,
              "end": 1771
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1772,
                "end": 1782
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1772,
              "end": 1782
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1783,
                "end": 1792
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1783,
              "end": 1792
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 1793,
                "end": 1800
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1793,
              "end": 1800
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 1801,
                "end": 1809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1801,
              "end": 1809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 1810,
                "end": 1820
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
                      "start": 1827,
                      "end": 1829
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1827,
                    "end": 1829
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 1834,
                      "end": 1851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1834,
                    "end": 1851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 1856,
                      "end": 1868
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1856,
                    "end": 1868
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 1873,
                      "end": 1883
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1873,
                    "end": 1883
                  }
                }
              ],
              "loc": {
                "start": 1821,
                "end": 1885
              }
            },
            "loc": {
              "start": 1810,
              "end": 1885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 1886,
                "end": 1897
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
                      "start": 1904,
                      "end": 1906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1904,
                    "end": 1906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 1911,
                      "end": 1925
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1911,
                    "end": 1925
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 1930,
                      "end": 1938
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1930,
                    "end": 1938
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 1943,
                      "end": 1952
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1943,
                    "end": 1952
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 1957,
                      "end": 1967
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1957,
                    "end": 1967
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 1972,
                      "end": 1977
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1972,
                    "end": 1977
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 1982,
                      "end": 1989
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1982,
                    "end": 1989
                  }
                }
              ],
              "loc": {
                "start": 1898,
                "end": 1991
              }
            },
            "loc": {
              "start": 1886,
              "end": 1991
            }
          }
        ],
        "loc": {
          "start": 1756,
          "end": 1993
        }
      },
      "loc": {
        "start": 1719,
        "end": 1993
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 2003,
          "end": 2016
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2020,
            "end": 2028
          }
        },
        "loc": {
          "start": 2020,
          "end": 2028
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
                "start": 2031,
                "end": 2037
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
                      "start": 2047,
                      "end": 2057
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2044,
                    "end": 2057
                  }
                }
              ],
              "loc": {
                "start": 2038,
                "end": 2059
              }
            },
            "loc": {
              "start": 2031,
              "end": 2059
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 2060,
                "end": 2070
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
                      "start": 2077,
                      "end": 2083
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
                            "start": 2094,
                            "end": 2096
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2094,
                          "end": 2096
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 2105,
                            "end": 2110
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2105,
                          "end": 2110
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2119,
                            "end": 2124
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2119,
                          "end": 2124
                        }
                      }
                    ],
                    "loc": {
                      "start": 2084,
                      "end": 2130
                    }
                  },
                  "loc": {
                    "start": 2077,
                    "end": 2130
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminderList",
                    "loc": {
                      "start": 2135,
                      "end": 2147
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
                            "start": 2158,
                            "end": 2160
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2158,
                          "end": 2160
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2169,
                            "end": 2179
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2169,
                          "end": 2179
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2188,
                            "end": 2198
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2188,
                          "end": 2198
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminders",
                          "loc": {
                            "start": 2207,
                            "end": 2216
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
                                  "start": 2231,
                                  "end": 2233
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2231,
                                "end": 2233
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2246,
                                  "end": 2256
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2246,
                                "end": 2256
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2269,
                                  "end": 2279
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2269,
                                "end": 2279
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2292,
                                  "end": 2296
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2292,
                                "end": 2296
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2309,
                                  "end": 2320
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2309,
                                "end": 2320
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 2333,
                                  "end": 2340
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2333,
                                "end": 2340
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2353,
                                  "end": 2358
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2353,
                                "end": 2358
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 2371,
                                  "end": 2381
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2371,
                                "end": 2381
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderItems",
                                "loc": {
                                  "start": 2394,
                                  "end": 2407
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
                                        "start": 2426,
                                        "end": 2428
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2426,
                                      "end": 2428
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2445,
                                        "end": 2455
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2445,
                                      "end": 2455
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 2472,
                                        "end": 2482
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2472,
                                      "end": 2482
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2499,
                                        "end": 2503
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2499,
                                      "end": 2503
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2520,
                                        "end": 2531
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2520,
                                      "end": 2531
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 2548,
                                        "end": 2555
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2548,
                                      "end": 2555
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 2572,
                                        "end": 2577
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2572,
                                      "end": 2577
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 2594,
                                        "end": 2604
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2594,
                                      "end": 2604
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2408,
                                  "end": 2618
                                }
                              },
                              "loc": {
                                "start": 2394,
                                "end": 2618
                              }
                            }
                          ],
                          "loc": {
                            "start": 2217,
                            "end": 2628
                          }
                        },
                        "loc": {
                          "start": 2207,
                          "end": 2628
                        }
                      }
                    ],
                    "loc": {
                      "start": 2148,
                      "end": 2634
                    }
                  },
                  "loc": {
                    "start": 2135,
                    "end": 2634
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 2639,
                      "end": 2651
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
                            "start": 2662,
                            "end": 2664
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2662,
                          "end": 2664
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2673,
                            "end": 2683
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2673,
                          "end": 2683
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 2692,
                            "end": 2704
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
                                  "start": 2719,
                                  "end": 2721
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2719,
                                "end": 2721
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 2734,
                                  "end": 2742
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2734,
                                "end": 2742
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2755,
                                  "end": 2766
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2755,
                                "end": 2766
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2779,
                                  "end": 2783
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2779,
                                "end": 2783
                              }
                            }
                          ],
                          "loc": {
                            "start": 2705,
                            "end": 2793
                          }
                        },
                        "loc": {
                          "start": 2692,
                          "end": 2793
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 2802,
                            "end": 2811
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
                                  "start": 2826,
                                  "end": 2828
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2826,
                                "end": 2828
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2841,
                                  "end": 2846
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2841,
                                "end": 2846
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 2859,
                                  "end": 2863
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2859,
                                "end": 2863
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 2876,
                                  "end": 2883
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2876,
                                "end": 2883
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 2896,
                                  "end": 2908
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
                                        "start": 2927,
                                        "end": 2929
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2927,
                                      "end": 2929
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 2946,
                                        "end": 2954
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2946,
                                      "end": 2954
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2971,
                                        "end": 2982
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2971,
                                      "end": 2982
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2999,
                                        "end": 3003
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2999,
                                      "end": 3003
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2909,
                                  "end": 3017
                                }
                              },
                              "loc": {
                                "start": 2896,
                                "end": 3017
                              }
                            }
                          ],
                          "loc": {
                            "start": 2812,
                            "end": 3027
                          }
                        },
                        "loc": {
                          "start": 2802,
                          "end": 3027
                        }
                      }
                    ],
                    "loc": {
                      "start": 2652,
                      "end": 3033
                    }
                  },
                  "loc": {
                    "start": 2639,
                    "end": 3033
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3038,
                      "end": 3040
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3038,
                    "end": 3040
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3045,
                      "end": 3049
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3045,
                    "end": 3049
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3054,
                      "end": 3065
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3054,
                    "end": 3065
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 3070,
                      "end": 3073
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
                            "start": 3084,
                            "end": 3093
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3084,
                          "end": 3093
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 3102,
                            "end": 3109
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3102,
                          "end": 3109
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 3118,
                            "end": 3127
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3118,
                          "end": 3127
                        }
                      }
                    ],
                    "loc": {
                      "start": 3074,
                      "end": 3133
                    }
                  },
                  "loc": {
                    "start": 3070,
                    "end": 3133
                  }
                }
              ],
              "loc": {
                "start": 2071,
                "end": 3135
              }
            },
            "loc": {
              "start": 2060,
              "end": 3135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetings",
              "loc": {
                "start": 3136,
                "end": 3144
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
                      "start": 3151,
                      "end": 3157
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
                            "start": 3171,
                            "end": 3181
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3168,
                          "end": 3181
                        }
                      }
                    ],
                    "loc": {
                      "start": 3158,
                      "end": 3187
                    }
                  },
                  "loc": {
                    "start": 3151,
                    "end": 3187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 3192,
                      "end": 3204
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
                            "start": 3215,
                            "end": 3217
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3215,
                          "end": 3217
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3226,
                            "end": 3234
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3226,
                          "end": 3234
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3243,
                            "end": 3254
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3243,
                          "end": 3254
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 3263,
                            "end": 3267
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3263,
                          "end": 3267
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3276,
                            "end": 3280
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3276,
                          "end": 3280
                        }
                      }
                    ],
                    "loc": {
                      "start": 3205,
                      "end": 3286
                    }
                  },
                  "loc": {
                    "start": 3192,
                    "end": 3286
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3291,
                      "end": 3293
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3291,
                    "end": 3293
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3298,
                      "end": 3308
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3298,
                    "end": 3308
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3313,
                      "end": 3323
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3313,
                    "end": 3323
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 3328,
                      "end": 3350
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3328,
                    "end": 3350
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnOrganizationProfile",
                    "loc": {
                      "start": 3355,
                      "end": 3380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3355,
                    "end": 3380
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 3385,
                      "end": 3397
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
                            "start": 3408,
                            "end": 3410
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3408,
                          "end": 3410
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 3419,
                            "end": 3430
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3419,
                          "end": 3430
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 3439,
                            "end": 3445
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3439,
                          "end": 3445
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 3454,
                            "end": 3466
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3454,
                          "end": 3466
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 3475,
                            "end": 3478
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
                                  "start": 3493,
                                  "end": 3506
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3493,
                                "end": 3506
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 3519,
                                  "end": 3528
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3519,
                                "end": 3528
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 3541,
                                  "end": 3552
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3541,
                                "end": 3552
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 3565,
                                  "end": 3574
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3565,
                                "end": 3574
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 3587,
                                  "end": 3596
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3587,
                                "end": 3596
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 3609,
                                  "end": 3616
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3609,
                                "end": 3616
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 3629,
                                  "end": 3641
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3629,
                                "end": 3641
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 3654,
                                  "end": 3662
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3654,
                                "end": 3662
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 3675,
                                  "end": 3689
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
                                        "start": 3708,
                                        "end": 3710
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3708,
                                      "end": 3710
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3727,
                                        "end": 3737
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3727,
                                      "end": 3737
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3754,
                                        "end": 3764
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3754,
                                      "end": 3764
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 3781,
                                        "end": 3788
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3781,
                                      "end": 3788
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3805,
                                        "end": 3816
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3805,
                                      "end": 3816
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3690,
                                  "end": 3830
                                }
                              },
                              "loc": {
                                "start": 3675,
                                "end": 3830
                              }
                            }
                          ],
                          "loc": {
                            "start": 3479,
                            "end": 3840
                          }
                        },
                        "loc": {
                          "start": 3475,
                          "end": 3840
                        }
                      }
                    ],
                    "loc": {
                      "start": 3398,
                      "end": 3846
                    }
                  },
                  "loc": {
                    "start": 3385,
                    "end": 3846
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 3851,
                      "end": 3868
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
                            "start": 3879,
                            "end": 3886
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
                                  "start": 3901,
                                  "end": 3903
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3901,
                                "end": 3903
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3916,
                                  "end": 3926
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3916,
                                "end": 3926
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3939,
                                  "end": 3949
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3939,
                                "end": 3949
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3962,
                                  "end": 3969
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3962,
                                "end": 3969
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3982,
                                  "end": 3993
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3982,
                                "end": 3993
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 4006,
                                  "end": 4011
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
                                        "start": 4030,
                                        "end": 4032
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4030,
                                      "end": 4032
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4049,
                                        "end": 4059
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4049,
                                      "end": 4059
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4076,
                                        "end": 4086
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4076,
                                      "end": 4086
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4103,
                                        "end": 4107
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4103,
                                      "end": 4107
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4124,
                                        "end": 4135
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4124,
                                      "end": 4135
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 4152,
                                        "end": 4164
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4152,
                                      "end": 4164
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "organization",
                                      "loc": {
                                        "start": 4181,
                                        "end": 4193
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
                                              "start": 4216,
                                              "end": 4218
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4216,
                                            "end": 4218
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 4239,
                                              "end": 4250
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4239,
                                            "end": 4250
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 4271,
                                              "end": 4277
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4271,
                                            "end": 4277
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 4298,
                                              "end": 4310
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4298,
                                            "end": 4310
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 4331,
                                              "end": 4334
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
                                                    "start": 4361,
                                                    "end": 4374
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4361,
                                                  "end": 4374
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 4399,
                                                    "end": 4408
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4399,
                                                  "end": 4408
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 4433,
                                                    "end": 4444
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4433,
                                                  "end": 4444
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 4469,
                                                    "end": 4478
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4469,
                                                  "end": 4478
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 4503,
                                                    "end": 4512
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4503,
                                                  "end": 4512
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 4537,
                                                    "end": 4544
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4537,
                                                  "end": 4544
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 4569,
                                                    "end": 4581
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4569,
                                                  "end": 4581
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 4606,
                                                    "end": 4614
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4606,
                                                  "end": 4614
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 4639,
                                                    "end": 4653
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
                                                          "start": 4684,
                                                          "end": 4686
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4684,
                                                        "end": 4686
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 4715,
                                                          "end": 4725
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4715,
                                                        "end": 4725
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 4754,
                                                          "end": 4764
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4754,
                                                        "end": 4764
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 4793,
                                                          "end": 4800
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4793,
                                                        "end": 4800
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 4829,
                                                          "end": 4840
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4829,
                                                        "end": 4840
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 4654,
                                                    "end": 4866
                                                  }
                                                },
                                                "loc": {
                                                  "start": 4639,
                                                  "end": 4866
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4335,
                                              "end": 4888
                                            }
                                          },
                                          "loc": {
                                            "start": 4331,
                                            "end": 4888
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4194,
                                        "end": 4906
                                      }
                                    },
                                    "loc": {
                                      "start": 4181,
                                      "end": 4906
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4923,
                                        "end": 4935
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
                                              "start": 4958,
                                              "end": 4960
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4958,
                                            "end": 4960
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4981,
                                              "end": 4989
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4981,
                                            "end": 4989
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5010,
                                              "end": 5021
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5010,
                                            "end": 5021
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4936,
                                        "end": 5039
                                      }
                                    },
                                    "loc": {
                                      "start": 4923,
                                      "end": 5039
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4012,
                                  "end": 5053
                                }
                              },
                              "loc": {
                                "start": 4006,
                                "end": 5053
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5066,
                                  "end": 5069
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
                                        "start": 5088,
                                        "end": 5097
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5088,
                                      "end": 5097
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 5114,
                                        "end": 5123
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5114,
                                      "end": 5123
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5070,
                                  "end": 5137
                                }
                              },
                              "loc": {
                                "start": 5066,
                                "end": 5137
                              }
                            }
                          ],
                          "loc": {
                            "start": 3887,
                            "end": 5147
                          }
                        },
                        "loc": {
                          "start": 3879,
                          "end": 5147
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5156,
                            "end": 5158
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5156,
                          "end": 5158
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5167,
                            "end": 5177
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5167,
                          "end": 5177
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5186,
                            "end": 5196
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5186,
                          "end": 5196
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5205,
                            "end": 5209
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5205,
                          "end": 5209
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 5218,
                            "end": 5229
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5218,
                          "end": 5229
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 5238,
                            "end": 5250
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5238,
                          "end": 5250
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "organization",
                          "loc": {
                            "start": 5259,
                            "end": 5271
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
                                  "start": 5286,
                                  "end": 5288
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5286,
                                "end": 5288
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 5301,
                                  "end": 5312
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5301,
                                "end": 5312
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 5325,
                                  "end": 5331
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5325,
                                "end": 5331
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 5344,
                                  "end": 5356
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5344,
                                "end": 5356
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5369,
                                  "end": 5372
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
                                        "start": 5391,
                                        "end": 5404
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5391,
                                      "end": 5404
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 5421,
                                        "end": 5430
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5421,
                                      "end": 5430
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 5447,
                                        "end": 5458
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5447,
                                      "end": 5458
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 5475,
                                        "end": 5484
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5475,
                                      "end": 5484
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 5501,
                                        "end": 5510
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5501,
                                      "end": 5510
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 5527,
                                        "end": 5534
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5527,
                                      "end": 5534
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 5551,
                                        "end": 5563
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5551,
                                      "end": 5563
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 5580,
                                        "end": 5588
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5580,
                                      "end": 5588
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 5605,
                                        "end": 5619
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
                                              "start": 5642,
                                              "end": 5644
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5642,
                                            "end": 5644
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5665,
                                              "end": 5675
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5665,
                                            "end": 5675
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 5696,
                                              "end": 5706
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5696,
                                            "end": 5706
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 5727,
                                              "end": 5734
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5727,
                                            "end": 5734
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 5755,
                                              "end": 5766
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5755,
                                            "end": 5766
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5620,
                                        "end": 5784
                                      }
                                    },
                                    "loc": {
                                      "start": 5605,
                                      "end": 5784
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5373,
                                  "end": 5798
                                }
                              },
                              "loc": {
                                "start": 5369,
                                "end": 5798
                              }
                            }
                          ],
                          "loc": {
                            "start": 5272,
                            "end": 5808
                          }
                        },
                        "loc": {
                          "start": 5259,
                          "end": 5808
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5817,
                            "end": 5829
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
                                  "start": 5844,
                                  "end": 5846
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5844,
                                "end": 5846
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5859,
                                  "end": 5867
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5859,
                                "end": 5867
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5880,
                                  "end": 5891
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5880,
                                "end": 5891
                              }
                            }
                          ],
                          "loc": {
                            "start": 5830,
                            "end": 5901
                          }
                        },
                        "loc": {
                          "start": 5817,
                          "end": 5901
                        }
                      }
                    ],
                    "loc": {
                      "start": 3869,
                      "end": 5907
                    }
                  },
                  "loc": {
                    "start": 3851,
                    "end": 5907
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 5912,
                      "end": 5926
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5912,
                    "end": 5926
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 5931,
                      "end": 5943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5931,
                    "end": 5943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5948,
                      "end": 5951
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
                            "start": 5962,
                            "end": 5971
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5962,
                          "end": 5971
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 5980,
                            "end": 5989
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5980,
                          "end": 5989
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5998,
                            "end": 6007
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5998,
                          "end": 6007
                        }
                      }
                    ],
                    "loc": {
                      "start": 5952,
                      "end": 6013
                    }
                  },
                  "loc": {
                    "start": 5948,
                    "end": 6013
                  }
                }
              ],
              "loc": {
                "start": 3145,
                "end": 6015
              }
            },
            "loc": {
              "start": 3136,
              "end": 6015
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 6016,
                "end": 6027
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
                      "start": 6034,
                      "end": 6048
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
                            "start": 6059,
                            "end": 6061
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6059,
                          "end": 6061
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 6070,
                            "end": 6080
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6070,
                          "end": 6080
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 6089,
                            "end": 6097
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6089,
                          "end": 6097
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6106,
                            "end": 6115
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6106,
                          "end": 6115
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6124,
                            "end": 6136
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6124,
                          "end": 6136
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6145,
                            "end": 6157
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6145,
                          "end": 6157
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6166,
                            "end": 6170
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
                                  "start": 6185,
                                  "end": 6187
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6185,
                                "end": 6187
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6200,
                                  "end": 6209
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6200,
                                "end": 6209
                              }
                            }
                          ],
                          "loc": {
                            "start": 6171,
                            "end": 6219
                          }
                        },
                        "loc": {
                          "start": 6166,
                          "end": 6219
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6228,
                            "end": 6240
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
                                  "start": 6255,
                                  "end": 6257
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6255,
                                "end": 6257
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6270,
                                  "end": 6278
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6270,
                                "end": 6278
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6291,
                                  "end": 6302
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6291,
                                "end": 6302
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 6315,
                                  "end": 6319
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6315,
                                "end": 6319
                              }
                            }
                          ],
                          "loc": {
                            "start": 6241,
                            "end": 6329
                          }
                        },
                        "loc": {
                          "start": 6228,
                          "end": 6329
                        }
                      }
                    ],
                    "loc": {
                      "start": 6049,
                      "end": 6335
                    }
                  },
                  "loc": {
                    "start": 6034,
                    "end": 6335
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6340,
                      "end": 6342
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6340,
                    "end": 6342
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6347,
                      "end": 6356
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6347,
                    "end": 6356
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6361,
                      "end": 6380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6361,
                    "end": 6380
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6385,
                      "end": 6400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6385,
                    "end": 6400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 6405,
                      "end": 6414
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6405,
                    "end": 6414
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6419,
                      "end": 6430
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6419,
                    "end": 6430
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 6435,
                      "end": 6446
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6435,
                    "end": 6446
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6451,
                      "end": 6455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6451,
                    "end": 6455
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 6460,
                      "end": 6466
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6460,
                    "end": 6466
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 6471,
                      "end": 6481
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6471,
                    "end": 6481
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 6486,
                      "end": 6498
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
                            "start": 6512,
                            "end": 6528
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6509,
                          "end": 6528
                        }
                      }
                    ],
                    "loc": {
                      "start": 6499,
                      "end": 6534
                    }
                  },
                  "loc": {
                    "start": 6486,
                    "end": 6534
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 6539,
                      "end": 6543
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
                            "start": 6557,
                            "end": 6565
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6554,
                          "end": 6565
                        }
                      }
                    ],
                    "loc": {
                      "start": 6544,
                      "end": 6571
                    }
                  },
                  "loc": {
                    "start": 6539,
                    "end": 6571
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6576,
                      "end": 6579
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
                            "start": 6590,
                            "end": 6599
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6590,
                          "end": 6599
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6608,
                            "end": 6617
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6608,
                          "end": 6617
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6626,
                            "end": 6633
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6626,
                          "end": 6633
                        }
                      }
                    ],
                    "loc": {
                      "start": 6580,
                      "end": 6639
                    }
                  },
                  "loc": {
                    "start": 6576,
                    "end": 6639
                  }
                }
              ],
              "loc": {
                "start": 6028,
                "end": 6641
              }
            },
            "loc": {
              "start": 6016,
              "end": 6641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 6642,
                "end": 6653
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
                      "start": 6660,
                      "end": 6674
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
                            "start": 6685,
                            "end": 6687
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6685,
                          "end": 6687
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 6696,
                            "end": 6706
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6696,
                          "end": 6706
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 6715,
                            "end": 6728
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6715,
                          "end": 6728
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 6737,
                            "end": 6747
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6737,
                          "end": 6747
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 6756,
                            "end": 6765
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6756,
                          "end": 6765
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 6774,
                            "end": 6782
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6774,
                          "end": 6782
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6791,
                            "end": 6800
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6791,
                          "end": 6800
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6809,
                            "end": 6813
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
                                  "start": 6828,
                                  "end": 6830
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6828,
                                "end": 6830
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 6843,
                                  "end": 6853
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6843,
                                "end": 6853
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6866,
                                  "end": 6875
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6866,
                                "end": 6875
                              }
                            }
                          ],
                          "loc": {
                            "start": 6814,
                            "end": 6885
                          }
                        },
                        "loc": {
                          "start": 6809,
                          "end": 6885
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6894,
                            "end": 6906
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
                                  "start": 6921,
                                  "end": 6923
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6921,
                                "end": 6923
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6936,
                                  "end": 6944
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6936,
                                "end": 6944
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6957,
                                  "end": 6968
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6957,
                                "end": 6968
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 6981,
                                  "end": 6993
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6981,
                                "end": 6993
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7006,
                                  "end": 7010
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7006,
                                "end": 7010
                              }
                            }
                          ],
                          "loc": {
                            "start": 6907,
                            "end": 7020
                          }
                        },
                        "loc": {
                          "start": 6894,
                          "end": 7020
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 7029,
                            "end": 7041
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7029,
                          "end": 7041
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 7050,
                            "end": 7062
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7050,
                          "end": 7062
                        }
                      }
                    ],
                    "loc": {
                      "start": 6675,
                      "end": 7068
                    }
                  },
                  "loc": {
                    "start": 6660,
                    "end": 7068
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7073,
                      "end": 7075
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7073,
                    "end": 7075
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7080,
                      "end": 7089
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7080,
                    "end": 7089
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 7094,
                      "end": 7113
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7094,
                    "end": 7113
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 7118,
                      "end": 7133
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7118,
                    "end": 7133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 7138,
                      "end": 7147
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7138,
                    "end": 7147
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 7152,
                      "end": 7163
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7152,
                    "end": 7163
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7168,
                      "end": 7179
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7168,
                    "end": 7179
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7184,
                      "end": 7188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7184,
                    "end": 7188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7193,
                      "end": 7199
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7193,
                    "end": 7199
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7204,
                      "end": 7214
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7204,
                    "end": 7214
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 7219,
                      "end": 7230
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7219,
                    "end": 7230
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 7235,
                      "end": 7254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7235,
                    "end": 7254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 7259,
                      "end": 7271
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
                            "start": 7285,
                            "end": 7301
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7282,
                          "end": 7301
                        }
                      }
                    ],
                    "loc": {
                      "start": 7272,
                      "end": 7307
                    }
                  },
                  "loc": {
                    "start": 7259,
                    "end": 7307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 7312,
                      "end": 7316
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
                            "start": 7330,
                            "end": 7338
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7327,
                          "end": 7338
                        }
                      }
                    ],
                    "loc": {
                      "start": 7317,
                      "end": 7344
                    }
                  },
                  "loc": {
                    "start": 7312,
                    "end": 7344
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7349,
                      "end": 7352
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
                            "start": 7363,
                            "end": 7372
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7363,
                          "end": 7372
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7381,
                            "end": 7390
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7381,
                          "end": 7390
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7399,
                            "end": 7406
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7399,
                          "end": 7406
                        }
                      }
                    ],
                    "loc": {
                      "start": 7353,
                      "end": 7412
                    }
                  },
                  "loc": {
                    "start": 7349,
                    "end": 7412
                  }
                }
              ],
              "loc": {
                "start": 6654,
                "end": 7414
              }
            },
            "loc": {
              "start": 6642,
              "end": 7414
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7415,
                "end": 7417
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7415,
              "end": 7417
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7418,
                "end": 7428
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7418,
              "end": 7428
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7429,
                "end": 7439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7429,
              "end": 7439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 7440,
                "end": 7449
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7440,
              "end": 7449
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 7450,
                "end": 7457
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7450,
              "end": 7457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 7458,
                "end": 7466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7458,
              "end": 7466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 7467,
                "end": 7477
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
                      "start": 7484,
                      "end": 7486
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7484,
                    "end": 7486
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 7491,
                      "end": 7508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7491,
                    "end": 7508
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 7513,
                      "end": 7525
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7513,
                    "end": 7525
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 7530,
                      "end": 7540
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7530,
                    "end": 7540
                  }
                }
              ],
              "loc": {
                "start": 7478,
                "end": 7542
              }
            },
            "loc": {
              "start": 7467,
              "end": 7542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 7543,
                "end": 7554
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
                      "start": 7561,
                      "end": 7563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7561,
                    "end": 7563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 7568,
                      "end": 7582
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7568,
                    "end": 7582
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 7587,
                      "end": 7595
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7587,
                    "end": 7595
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 7600,
                      "end": 7609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7600,
                    "end": 7609
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 7614,
                      "end": 7624
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7614,
                    "end": 7624
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 7629,
                      "end": 7634
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7629,
                    "end": 7634
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 7639,
                      "end": 7646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7639,
                    "end": 7646
                  }
                }
              ],
              "loc": {
                "start": 7555,
                "end": 7648
              }
            },
            "loc": {
              "start": 7543,
              "end": 7648
            }
          }
        ],
        "loc": {
          "start": 2029,
          "end": 7650
        }
      },
      "loc": {
        "start": 1994,
        "end": 7650
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7660,
          "end": 7668
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7672,
            "end": 7676
          }
        },
        "loc": {
          "start": 7672,
          "end": 7676
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
                "start": 7679,
                "end": 7681
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7679,
              "end": 7681
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7682,
                "end": 7692
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7682,
              "end": 7692
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7693,
                "end": 7703
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7693,
              "end": 7703
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7704,
                "end": 7715
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7704,
              "end": 7715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7716,
                "end": 7722
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7716,
              "end": 7722
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7723,
                "end": 7728
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7723,
              "end": 7728
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7729,
                "end": 7749
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7729,
              "end": 7749
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7750,
                "end": 7754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7750,
              "end": 7754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7755,
                "end": 7767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7755,
              "end": 7767
            }
          }
        ],
        "loc": {
          "start": 7677,
          "end": 7769
        }
      },
      "loc": {
        "start": 7651,
        "end": 7769
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
        "start": 7777,
        "end": 7781
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
              "start": 7783,
              "end": 7788
            }
          },
          "loc": {
            "start": 7782,
            "end": 7788
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
                "start": 7790,
                "end": 7799
              }
            },
            "loc": {
              "start": 7790,
              "end": 7799
            }
          },
          "loc": {
            "start": 7790,
            "end": 7800
          }
        },
        "directives": [],
        "loc": {
          "start": 7782,
          "end": 7800
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
              "start": 7806,
              "end": 7810
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7811,
                  "end": 7816
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7819,
                    "end": 7824
                  }
                },
                "loc": {
                  "start": 7818,
                  "end": 7824
                }
              },
              "loc": {
                "start": 7811,
                "end": 7824
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
                    "start": 7832,
                    "end": 7843
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
                          "start": 7857,
                          "end": 7870
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7854,
                        "end": 7870
                      }
                    }
                  ],
                  "loc": {
                    "start": 7844,
                    "end": 7876
                  }
                },
                "loc": {
                  "start": 7832,
                  "end": 7876
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 7881,
                    "end": 7890
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
                          "start": 7904,
                          "end": 7917
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7901,
                        "end": 7917
                      }
                    }
                  ],
                  "loc": {
                    "start": 7891,
                    "end": 7923
                  }
                },
                "loc": {
                  "start": 7881,
                  "end": 7923
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 7928,
                    "end": 7937
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
                          "start": 7951,
                          "end": 7964
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7948,
                        "end": 7964
                      }
                    }
                  ],
                  "loc": {
                    "start": 7938,
                    "end": 7970
                  }
                },
                "loc": {
                  "start": 7928,
                  "end": 7970
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 7975,
                    "end": 7984
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
                          "start": 7998,
                          "end": 8011
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7995,
                        "end": 8011
                      }
                    }
                  ],
                  "loc": {
                    "start": 7985,
                    "end": 8017
                  }
                },
                "loc": {
                  "start": 7975,
                  "end": 8017
                }
              }
            ],
            "loc": {
              "start": 7826,
              "end": 8021
            }
          },
          "loc": {
            "start": 7806,
            "end": 8021
          }
        }
      ],
      "loc": {
        "start": 7802,
        "end": 8023
      }
    },
    "loc": {
      "start": 7771,
      "end": 8023
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
