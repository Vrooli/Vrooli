export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 8419,
          "end": 8423
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8424,
              "end": 8429
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8432,
                "end": 8437
              }
            },
            "loc": {
              "start": 8431,
              "end": 8437
            }
          },
          "loc": {
            "start": 8424,
            "end": 8437
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
                "start": 8445,
                "end": 8456
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
                      "start": 8470,
                      "end": 8483
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8467,
                    "end": 8483
                  }
                }
              ],
              "loc": {
                "start": 8457,
                "end": 8489
              }
            },
            "loc": {
              "start": 8445,
              "end": 8489
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 8494,
                "end": 8503
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
                      "start": 8517,
                      "end": 8530
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8514,
                    "end": 8530
                  }
                }
              ],
              "loc": {
                "start": 8504,
                "end": 8536
              }
            },
            "loc": {
              "start": 8494,
              "end": 8536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 8541,
                "end": 8550
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
                      "start": 8564,
                      "end": 8577
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8561,
                    "end": 8577
                  }
                }
              ],
              "loc": {
                "start": 8551,
                "end": 8583
              }
            },
            "loc": {
              "start": 8541,
              "end": 8583
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 8588,
                "end": 8597
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
                      "start": 8611,
                      "end": 8624
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8608,
                    "end": 8624
                  }
                }
              ],
              "loc": {
                "start": 8598,
                "end": 8630
              }
            },
            "loc": {
              "start": 8588,
              "end": 8630
            }
          }
        ],
        "loc": {
          "start": 8439,
          "end": 8634
        }
      },
      "loc": {
        "start": 8419,
        "end": 8634
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
                "value": "Team",
                "loc": {
                  "start": 88,
                  "end": 92
                }
              },
              "loc": {
                "start": 88,
                "end": 92
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
                    "value": "Team_nav",
                    "loc": {
                      "start": 106,
                      "end": 114
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 103,
                    "end": 114
                  }
                }
              ],
              "loc": {
                "start": 93,
                "end": 120
              }
            },
            "loc": {
              "start": 81,
              "end": 120
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
                  "start": 132,
                  "end": 136
                }
              },
              "loc": {
                "start": 132,
                "end": 136
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
                      "start": 150,
                      "end": 158
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 147,
                    "end": 158
                  }
                }
              ],
              "loc": {
                "start": 137,
                "end": 164
              }
            },
            "loc": {
              "start": 125,
              "end": 164
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 166
        }
      },
      "loc": {
        "start": 69,
        "end": 166
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 167,
          "end": 170
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
                "start": 177,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 177,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 191,
                "end": 200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 200
            }
          }
        ],
        "loc": {
          "start": 171,
          "end": 202
        }
      },
      "loc": {
        "start": 167,
        "end": 202
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 242,
          "end": 244
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 242,
        "end": 244
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 245,
          "end": 255
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 245,
        "end": 255
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 256,
          "end": 266
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 256,
        "end": 266
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 267,
          "end": 271
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 267,
        "end": 271
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "description",
        "loc": {
          "start": 272,
          "end": 283
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 272,
        "end": 283
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "dueDate",
        "loc": {
          "start": 284,
          "end": 291
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 284,
        "end": 291
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 292,
          "end": 297
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 292,
        "end": 297
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isComplete",
        "loc": {
          "start": 298,
          "end": 308
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 298,
        "end": 308
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderItems",
        "loc": {
          "start": 309,
          "end": 322
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
                "start": 329,
                "end": 331
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 329,
              "end": 331
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 336,
                "end": 346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 336,
              "end": 346
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 351,
                "end": 361
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 351,
              "end": 361
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 366,
                "end": 370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 366,
              "end": 370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 375,
                "end": 386
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 375,
              "end": 386
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 391,
                "end": 398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 391,
              "end": 398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 403,
                "end": 408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 403,
              "end": 408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 413,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 413,
              "end": 423
            }
          }
        ],
        "loc": {
          "start": 323,
          "end": 425
        }
      },
      "loc": {
        "start": 309,
        "end": 425
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderList",
        "loc": {
          "start": 426,
          "end": 438
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
                "start": 445,
                "end": 447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 445,
              "end": 447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 452,
                "end": 462
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 452,
              "end": 462
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 467,
                "end": 477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 467,
              "end": 477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusMode",
              "loc": {
                "start": 482,
                "end": 491
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
                      "start": 502,
                      "end": 504
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 502,
                    "end": 504
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 513,
                      "end": 517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 513,
                    "end": 517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 526,
                      "end": 537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 526,
                    "end": 537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 546,
                      "end": 549
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
                            "start": 564,
                            "end": 573
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 564,
                          "end": 573
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 586,
                            "end": 593
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 586,
                          "end": 593
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 606,
                            "end": 615
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 606,
                          "end": 615
                        }
                      }
                    ],
                    "loc": {
                      "start": 550,
                      "end": 625
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 625
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 634,
                      "end": 640
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
                            "start": 655,
                            "end": 657
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 655,
                          "end": 657
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 670,
                            "end": 675
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 670,
                          "end": 675
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 688,
                            "end": 693
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 688,
                          "end": 693
                        }
                      }
                    ],
                    "loc": {
                      "start": 641,
                      "end": 703
                    }
                  },
                  "loc": {
                    "start": 634,
                    "end": 703
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 712,
                      "end": 724
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
                            "start": 739,
                            "end": 741
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 739,
                          "end": 741
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 754,
                            "end": 764
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 754,
                          "end": 764
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 777,
                            "end": 789
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
                                  "start": 808,
                                  "end": 810
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 808,
                                "end": 810
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 827,
                                  "end": 835
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 827,
                                "end": 835
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 852,
                                  "end": 863
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 852,
                                "end": 863
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 880,
                                  "end": 884
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 880,
                                "end": 884
                              }
                            }
                          ],
                          "loc": {
                            "start": 790,
                            "end": 898
                          }
                        },
                        "loc": {
                          "start": 777,
                          "end": 898
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 911,
                            "end": 920
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
                                  "start": 939,
                                  "end": 941
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 939,
                                "end": 941
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 958,
                                  "end": 963
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 958,
                                "end": 963
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 980,
                                  "end": 984
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 980,
                                "end": 984
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 1001,
                                  "end": 1008
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1001,
                                "end": 1008
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 1025,
                                  "end": 1037
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
                                        "start": 1060,
                                        "end": 1062
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1060,
                                      "end": 1062
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 1083,
                                        "end": 1091
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1083,
                                      "end": 1091
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 1112,
                                        "end": 1123
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1112,
                                      "end": 1123
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 1144,
                                        "end": 1148
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1144,
                                      "end": 1148
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1038,
                                  "end": 1166
                                }
                              },
                              "loc": {
                                "start": 1025,
                                "end": 1166
                              }
                            }
                          ],
                          "loc": {
                            "start": 921,
                            "end": 1180
                          }
                        },
                        "loc": {
                          "start": 911,
                          "end": 1180
                        }
                      }
                    ],
                    "loc": {
                      "start": 725,
                      "end": 1190
                    }
                  },
                  "loc": {
                    "start": 712,
                    "end": 1190
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 1199,
                      "end": 1207
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
                            "start": 1225,
                            "end": 1240
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1222,
                          "end": 1240
                        }
                      }
                    ],
                    "loc": {
                      "start": 1208,
                      "end": 1250
                    }
                  },
                  "loc": {
                    "start": 1199,
                    "end": 1250
                  }
                }
              ],
              "loc": {
                "start": 492,
                "end": 1256
              }
            },
            "loc": {
              "start": 482,
              "end": 1256
            }
          }
        ],
        "loc": {
          "start": 439,
          "end": 1258
        }
      },
      "loc": {
        "start": 426,
        "end": 1258
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1298,
          "end": 1300
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1298,
        "end": 1300
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1301,
          "end": 1306
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1301,
        "end": 1306
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 1307,
          "end": 1311
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1307,
        "end": 1311
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 1312,
          "end": 1319
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1312,
        "end": 1319
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1320,
          "end": 1332
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
                "start": 1339,
                "end": 1341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1339,
              "end": 1341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1346,
                "end": 1354
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1346,
              "end": 1354
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1359,
                "end": 1370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1359,
              "end": 1370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1375,
                "end": 1379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1375,
              "end": 1379
            }
          }
        ],
        "loc": {
          "start": 1333,
          "end": 1381
        }
      },
      "loc": {
        "start": 1320,
        "end": 1381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1423,
          "end": 1425
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1423,
        "end": 1425
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1426,
          "end": 1436
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1426,
        "end": 1436
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1437,
          "end": 1447
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1437,
        "end": 1447
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 1448,
          "end": 1457
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1448,
        "end": 1457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 1458,
          "end": 1465
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1458,
        "end": 1465
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 1466,
          "end": 1474
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1466,
        "end": 1474
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 1475,
          "end": 1485
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
                "start": 1492,
                "end": 1494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1492,
              "end": 1494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 1499,
                "end": 1516
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1499,
              "end": 1516
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 1521,
                "end": 1533
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1521,
              "end": 1533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 1538,
                "end": 1548
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1538,
              "end": 1548
            }
          }
        ],
        "loc": {
          "start": 1486,
          "end": 1550
        }
      },
      "loc": {
        "start": 1475,
        "end": 1550
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 1551,
          "end": 1562
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
                "start": 1569,
                "end": 1571
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1569,
              "end": 1571
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 1576,
                "end": 1590
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1576,
              "end": 1590
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 1595,
                "end": 1603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1595,
              "end": 1603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 1608,
                "end": 1617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1608,
              "end": 1617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 1622,
                "end": 1632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1622,
              "end": 1632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 1637,
                "end": 1642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1637,
              "end": 1642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
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
          }
        ],
        "loc": {
          "start": 1563,
          "end": 1656
        }
      },
      "loc": {
        "start": 1551,
        "end": 1656
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1696,
          "end": 1698
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1696,
        "end": 1698
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1699,
          "end": 1709
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1699,
        "end": 1709
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1710,
          "end": 1720
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1710,
        "end": 1720
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 1721,
          "end": 1730
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1721,
        "end": 1730
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 1731,
          "end": 1738
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1731,
        "end": 1738
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 1739,
          "end": 1747
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1739,
        "end": 1747
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 1748,
          "end": 1758
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
                "start": 1765,
                "end": 1767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1765,
              "end": 1767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 1772,
                "end": 1789
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1772,
              "end": 1789
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 1794,
                "end": 1806
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1794,
              "end": 1806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 1811,
                "end": 1821
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1811,
              "end": 1821
            }
          }
        ],
        "loc": {
          "start": 1759,
          "end": 1823
        }
      },
      "loc": {
        "start": 1748,
        "end": 1823
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 1824,
          "end": 1835
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
                "start": 1842,
                "end": 1844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1842,
              "end": 1844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 1849,
                "end": 1863
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1849,
              "end": 1863
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 1868,
                "end": 1876
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1868,
              "end": 1876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 1881,
                "end": 1890
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1881,
              "end": 1890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 1895,
                "end": 1905
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1895,
              "end": 1905
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 1910,
                "end": 1915
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1910,
              "end": 1915
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 1920,
                "end": 1927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1920,
              "end": 1927
            }
          }
        ],
        "loc": {
          "start": 1836,
          "end": 1929
        }
      },
      "loc": {
        "start": 1824,
        "end": 1929
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1930,
          "end": 1936
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
                "start": 1946,
                "end": 1956
              }
            },
            "directives": [],
            "loc": {
              "start": 1943,
              "end": 1956
            }
          }
        ],
        "loc": {
          "start": 1937,
          "end": 1958
        }
      },
      "loc": {
        "start": 1930,
        "end": 1958
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModes",
        "loc": {
          "start": 1959,
          "end": 1969
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
                "start": 1976,
                "end": 1978
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1976,
              "end": 1978
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1983,
                "end": 1987
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1983,
              "end": 1987
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1992,
                "end": 2003
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1992,
              "end": 2003
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2008,
                "end": 2011
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
                      "start": 2022,
                      "end": 2031
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2022,
                    "end": 2031
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2040,
                      "end": 2047
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2040,
                    "end": 2047
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2056,
                      "end": 2065
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2056,
                    "end": 2065
                  }
                }
              ],
              "loc": {
                "start": 2012,
                "end": 2071
              }
            },
            "loc": {
              "start": 2008,
              "end": 2071
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2076,
                "end": 2082
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
                      "start": 2093,
                      "end": 2095
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2093,
                    "end": 2095
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "color",
                    "loc": {
                      "start": 2104,
                      "end": 2109
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2104,
                    "end": 2109
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 2118,
                      "end": 2123
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2118,
                    "end": 2123
                  }
                }
              ],
              "loc": {
                "start": 2083,
                "end": 2129
              }
            },
            "loc": {
              "start": 2076,
              "end": 2129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 2134,
                "end": 2146
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
                      "start": 2157,
                      "end": 2159
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2157,
                    "end": 2159
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2168,
                      "end": 2178
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2168,
                    "end": 2178
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2187,
                      "end": 2197
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2187,
                    "end": 2197
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminders",
                    "loc": {
                      "start": 2206,
                      "end": 2215
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
                            "start": 2230,
                            "end": 2232
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2230,
                          "end": 2232
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2245,
                            "end": 2255
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2245,
                          "end": 2255
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2268,
                            "end": 2278
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2268,
                          "end": 2278
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2291,
                            "end": 2295
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2291,
                          "end": 2295
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2308,
                            "end": 2319
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2308,
                          "end": 2319
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dueDate",
                          "loc": {
                            "start": 2332,
                            "end": 2339
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2332,
                          "end": 2339
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 2352,
                            "end": 2357
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2352,
                          "end": 2357
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 2370,
                            "end": 2380
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2370,
                          "end": 2380
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderItems",
                          "loc": {
                            "start": 2393,
                            "end": 2406
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
                                  "start": 2425,
                                  "end": 2427
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2425,
                                "end": 2427
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2444,
                                  "end": 2454
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2444,
                                "end": 2454
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2471,
                                  "end": 2481
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2471,
                                "end": 2481
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2498,
                                  "end": 2502
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2498,
                                "end": 2502
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2519,
                                  "end": 2530
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2519,
                                "end": 2530
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 2547,
                                  "end": 2554
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2547,
                                "end": 2554
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2571,
                                  "end": 2576
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2571,
                                "end": 2576
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 2593,
                                  "end": 2603
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2593,
                                "end": 2603
                              }
                            }
                          ],
                          "loc": {
                            "start": 2407,
                            "end": 2617
                          }
                        },
                        "loc": {
                          "start": 2393,
                          "end": 2617
                        }
                      }
                    ],
                    "loc": {
                      "start": 2216,
                      "end": 2627
                    }
                  },
                  "loc": {
                    "start": 2206,
                    "end": 2627
                  }
                }
              ],
              "loc": {
                "start": 2147,
                "end": 2633
              }
            },
            "loc": {
              "start": 2134,
              "end": 2633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resourceList",
              "loc": {
                "start": 2638,
                "end": 2650
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
                      "start": 2661,
                      "end": 2663
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2661,
                    "end": 2663
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2672,
                      "end": 2682
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2672,
                    "end": 2682
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2691,
                      "end": 2703
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
                            "start": 2718,
                            "end": 2720
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2718,
                          "end": 2720
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2733,
                            "end": 2741
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2733,
                          "end": 2741
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2754,
                            "end": 2765
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2754,
                          "end": 2765
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2778,
                            "end": 2782
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2778,
                          "end": 2782
                        }
                      }
                    ],
                    "loc": {
                      "start": 2704,
                      "end": 2792
                    }
                  },
                  "loc": {
                    "start": 2691,
                    "end": 2792
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resources",
                    "loc": {
                      "start": 2801,
                      "end": 2810
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
                            "start": 2825,
                            "end": 2827
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2825,
                          "end": 2827
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 2840,
                            "end": 2845
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2840,
                          "end": 2845
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 2858,
                            "end": 2862
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2858,
                          "end": 2862
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "usedFor",
                          "loc": {
                            "start": 2875,
                            "end": 2882
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2875,
                          "end": 2882
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 2895,
                            "end": 2907
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
                                  "start": 2926,
                                  "end": 2928
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2926,
                                "end": 2928
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 2945,
                                  "end": 2953
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2945,
                                "end": 2953
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2970,
                                  "end": 2981
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2970,
                                "end": 2981
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2998,
                                  "end": 3002
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2998,
                                "end": 3002
                              }
                            }
                          ],
                          "loc": {
                            "start": 2908,
                            "end": 3016
                          }
                        },
                        "loc": {
                          "start": 2895,
                          "end": 3016
                        }
                      }
                    ],
                    "loc": {
                      "start": 2811,
                      "end": 3026
                    }
                  },
                  "loc": {
                    "start": 2801,
                    "end": 3026
                  }
                }
              ],
              "loc": {
                "start": 2651,
                "end": 3032
              }
            },
            "loc": {
              "start": 2638,
              "end": 3032
            }
          }
        ],
        "loc": {
          "start": 1970,
          "end": 3034
        }
      },
      "loc": {
        "start": 1959,
        "end": 3034
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetings",
        "loc": {
          "start": 3035,
          "end": 3043
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
                "start": 3050,
                "end": 3052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3050,
              "end": 3052
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3057,
                "end": 3067
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3057,
              "end": 3067
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3072,
                "end": 3082
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3072,
              "end": 3082
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 3087,
                "end": 3109
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3087,
              "end": 3109
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnTeamProfile",
              "loc": {
                "start": 3114,
                "end": 3131
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3114,
              "end": 3131
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "team",
              "loc": {
                "start": 3136,
                "end": 3140
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
                      "start": 3151,
                      "end": 3153
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3151,
                    "end": 3153
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3162,
                      "end": 3173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3162,
                    "end": 3173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3182,
                      "end": 3188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3182,
                    "end": 3188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3197,
                      "end": 3209
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3197,
                    "end": 3209
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 3218,
                      "end": 3221
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
                            "start": 3236,
                            "end": 3249
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3236,
                          "end": 3249
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 3262,
                            "end": 3271
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3262,
                          "end": 3271
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 3284,
                            "end": 3295
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3284,
                          "end": 3295
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 3308,
                            "end": 3317
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3308,
                          "end": 3317
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 3330,
                            "end": 3339
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3330,
                          "end": 3339
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 3352,
                            "end": 3359
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3352,
                          "end": 3359
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 3372,
                            "end": 3384
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3372,
                          "end": 3384
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 3397,
                            "end": 3405
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3397,
                          "end": 3405
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 3418,
                            "end": 3432
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
                                  "start": 3451,
                                  "end": 3453
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3451,
                                "end": 3453
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3470,
                                  "end": 3480
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3470,
                                "end": 3480
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3497,
                                  "end": 3507
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3497,
                                "end": 3507
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3524,
                                  "end": 3531
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3524,
                                "end": 3531
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3548,
                                  "end": 3559
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3548,
                                "end": 3559
                              }
                            }
                          ],
                          "loc": {
                            "start": 3433,
                            "end": 3573
                          }
                        },
                        "loc": {
                          "start": 3418,
                          "end": 3573
                        }
                      }
                    ],
                    "loc": {
                      "start": 3222,
                      "end": 3583
                    }
                  },
                  "loc": {
                    "start": 3218,
                    "end": 3583
                  }
                }
              ],
              "loc": {
                "start": 3141,
                "end": 3589
              }
            },
            "loc": {
              "start": 3136,
              "end": 3589
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 3594,
                "end": 3611
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
                      "start": 3622,
                      "end": 3624
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3622,
                    "end": 3624
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3633,
                      "end": 3643
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3633,
                    "end": 3643
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3652,
                      "end": 3662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3652,
                    "end": 3662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3671,
                      "end": 3675
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3671,
                    "end": 3675
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 3684,
                      "end": 3695
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3684,
                    "end": 3695
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 3704,
                      "end": 3716
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3704,
                    "end": 3716
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "team",
                    "loc": {
                      "start": 3725,
                      "end": 3729
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
                            "start": 3744,
                            "end": 3746
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3744,
                          "end": 3746
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 3759,
                            "end": 3770
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3759,
                          "end": 3770
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 3783,
                            "end": 3789
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3783,
                          "end": 3789
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 3802,
                            "end": 3814
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3802,
                          "end": 3814
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 3827,
                            "end": 3830
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
                                  "start": 3849,
                                  "end": 3862
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3849,
                                "end": 3862
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 3879,
                                  "end": 3888
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3879,
                                "end": 3888
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 3905,
                                  "end": 3916
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3905,
                                "end": 3916
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 3933,
                                  "end": 3942
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3933,
                                "end": 3942
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 3959,
                                  "end": 3968
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3959,
                                "end": 3968
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 3985,
                                  "end": 3992
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3985,
                                "end": 3992
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 4009,
                                  "end": 4021
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4009,
                                "end": 4021
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 4038,
                                  "end": 4046
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4038,
                                "end": 4046
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 4063,
                                  "end": 4077
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
                                        "start": 4100,
                                        "end": 4102
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4100,
                                      "end": 4102
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4123,
                                        "end": 4133
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4123,
                                      "end": 4133
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4154,
                                        "end": 4164
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4154,
                                      "end": 4164
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 4185,
                                        "end": 4192
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4185,
                                      "end": 4192
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4213,
                                        "end": 4224
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4213,
                                      "end": 4224
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4078,
                                  "end": 4242
                                }
                              },
                              "loc": {
                                "start": 4063,
                                "end": 4242
                              }
                            }
                          ],
                          "loc": {
                            "start": 3831,
                            "end": 4256
                          }
                        },
                        "loc": {
                          "start": 3827,
                          "end": 4256
                        }
                      }
                    ],
                    "loc": {
                      "start": 3730,
                      "end": 4266
                    }
                  },
                  "loc": {
                    "start": 3725,
                    "end": 4266
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 4275,
                      "end": 4287
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
                            "start": 4302,
                            "end": 4304
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4302,
                          "end": 4304
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4317,
                            "end": 4325
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4317,
                          "end": 4325
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4338,
                            "end": 4349
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4338,
                          "end": 4349
                        }
                      }
                    ],
                    "loc": {
                      "start": 4288,
                      "end": 4359
                    }
                  },
                  "loc": {
                    "start": 4275,
                    "end": 4359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "members",
                    "loc": {
                      "start": 4368,
                      "end": 4375
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
                            "start": 4390,
                            "end": 4392
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4390,
                          "end": 4392
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4405,
                            "end": 4415
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4405,
                          "end": 4415
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4428,
                            "end": 4438
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4428,
                          "end": 4438
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 4451,
                            "end": 4458
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4451,
                          "end": 4458
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 4471,
                            "end": 4482
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4471,
                          "end": 4482
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 4495,
                            "end": 4500
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
                                  "start": 4519,
                                  "end": 4521
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4519,
                                "end": 4521
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4538,
                                  "end": 4548
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4538,
                                "end": 4548
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4565,
                                  "end": 4575
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4565,
                                "end": 4575
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4592,
                                  "end": 4596
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4592,
                                "end": 4596
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4613,
                                  "end": 4624
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4613,
                                "end": 4624
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 4641,
                                  "end": 4653
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4641,
                                "end": 4653
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "team",
                                "loc": {
                                  "start": 4670,
                                  "end": 4674
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
                                        "start": 4697,
                                        "end": 4699
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4697,
                                      "end": 4699
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 4720,
                                        "end": 4731
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4720,
                                      "end": 4731
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 4752,
                                        "end": 4758
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4752,
                                      "end": 4758
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 4779,
                                        "end": 4791
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4779,
                                      "end": 4791
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 4812,
                                        "end": 4815
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
                                              "start": 4842,
                                              "end": 4855
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4842,
                                            "end": 4855
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 4880,
                                              "end": 4889
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4880,
                                            "end": 4889
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 4914,
                                              "end": 4925
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4914,
                                            "end": 4925
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 4950,
                                              "end": 4959
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4950,
                                            "end": 4959
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 4984,
                                              "end": 4993
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4984,
                                            "end": 4993
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 5018,
                                              "end": 5025
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5018,
                                            "end": 5025
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5050,
                                              "end": 5062
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5050,
                                            "end": 5062
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 5087,
                                              "end": 5095
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5087,
                                            "end": 5095
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 5120,
                                              "end": 5134
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
                                                    "start": 5165,
                                                    "end": 5167
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5165,
                                                  "end": 5167
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 5196,
                                                    "end": 5206
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5196,
                                                  "end": 5206
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 5235,
                                                    "end": 5245
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5235,
                                                  "end": 5245
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 5274,
                                                    "end": 5281
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5274,
                                                  "end": 5281
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 5310,
                                                    "end": 5321
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5310,
                                                  "end": 5321
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5135,
                                              "end": 5347
                                            }
                                          },
                                          "loc": {
                                            "start": 5120,
                                            "end": 5347
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4816,
                                        "end": 5369
                                      }
                                    },
                                    "loc": {
                                      "start": 4812,
                                      "end": 5369
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4675,
                                  "end": 5387
                                }
                              },
                              "loc": {
                                "start": 4670,
                                "end": 5387
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 5404,
                                  "end": 5416
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
                                        "start": 5439,
                                        "end": 5441
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5439,
                                      "end": 5441
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 5462,
                                        "end": 5470
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5462,
                                      "end": 5470
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5491,
                                        "end": 5502
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5491,
                                      "end": 5502
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5417,
                                  "end": 5520
                                }
                              },
                              "loc": {
                                "start": 5404,
                                "end": 5520
                              }
                            }
                          ],
                          "loc": {
                            "start": 4501,
                            "end": 5534
                          }
                        },
                        "loc": {
                          "start": 4495,
                          "end": 5534
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 5547,
                            "end": 5550
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
                                  "start": 5569,
                                  "end": 5578
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5569,
                                "end": 5578
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 5595,
                                  "end": 5604
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5595,
                                "end": 5604
                              }
                            }
                          ],
                          "loc": {
                            "start": 5551,
                            "end": 5618
                          }
                        },
                        "loc": {
                          "start": 5547,
                          "end": 5618
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "user",
                          "loc": {
                            "start": 5631,
                            "end": 5635
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
                                  "start": 5654,
                                  "end": 5656
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5654,
                                "end": 5656
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 5673,
                                  "end": 5683
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5673,
                                "end": 5683
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 5700,
                                  "end": 5710
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5700,
                                "end": 5710
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 5727,
                                  "end": 5738
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5727,
                                "end": 5738
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 5755,
                                  "end": 5761
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5755,
                                "end": 5761
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBot",
                                "loc": {
                                  "start": 5778,
                                  "end": 5783
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5778,
                                "end": 5783
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBotDepictingPerson",
                                "loc": {
                                  "start": 5800,
                                  "end": 5820
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5800,
                                "end": 5820
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 5837,
                                  "end": 5841
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5837,
                                "end": 5841
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 5858,
                                  "end": 5870
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5858,
                                "end": 5870
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bookmarks",
                                "loc": {
                                  "start": 5887,
                                  "end": 5896
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5887,
                                "end": 5896
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reportsReceivedCount",
                                "loc": {
                                  "start": 5913,
                                  "end": 5933
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5913,
                                "end": 5933
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5950,
                                  "end": 5953
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
                                        "start": 5976,
                                        "end": 5985
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5976,
                                      "end": 5985
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 6006,
                                        "end": 6015
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6006,
                                      "end": 6015
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 6036,
                                        "end": 6045
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6036,
                                      "end": 6045
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 6066,
                                        "end": 6078
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6066,
                                      "end": 6078
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 6099,
                                        "end": 6107
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6099,
                                      "end": 6107
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5954,
                                  "end": 6125
                                }
                              },
                              "loc": {
                                "start": 5950,
                                "end": 6125
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 6142,
                                  "end": 6154
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
                                        "start": 6177,
                                        "end": 6179
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6177,
                                      "end": 6179
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 6200,
                                        "end": 6208
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6200,
                                      "end": 6208
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bio",
                                      "loc": {
                                        "start": 6229,
                                        "end": 6232
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6229,
                                      "end": 6232
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6155,
                                  "end": 6250
                                }
                              },
                              "loc": {
                                "start": 6142,
                                "end": 6250
                              }
                            }
                          ],
                          "loc": {
                            "start": 5636,
                            "end": 6264
                          }
                        },
                        "loc": {
                          "start": 5631,
                          "end": 6264
                        }
                      }
                    ],
                    "loc": {
                      "start": 4376,
                      "end": 6274
                    }
                  },
                  "loc": {
                    "start": 4368,
                    "end": 6274
                  }
                }
              ],
              "loc": {
                "start": 3612,
                "end": 6280
              }
            },
            "loc": {
              "start": 3594,
              "end": 6280
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 6285,
                "end": 6299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6285,
              "end": 6299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 6304,
                "end": 6316
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6304,
              "end": 6316
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6321,
                "end": 6324
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
                      "start": 6335,
                      "end": 6344
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6335,
                    "end": 6344
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 6353,
                      "end": 6362
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6353,
                    "end": 6362
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6371,
                      "end": 6380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6371,
                    "end": 6380
                  }
                }
              ],
              "loc": {
                "start": 6325,
                "end": 6386
              }
            },
            "loc": {
              "start": 6321,
              "end": 6386
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6391,
                "end": 6397
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
                      "start": 6411,
                      "end": 6421
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6408,
                    "end": 6421
                  }
                }
              ],
              "loc": {
                "start": 6398,
                "end": 6427
              }
            },
            "loc": {
              "start": 6391,
              "end": 6427
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6432,
                "end": 6444
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
                      "start": 6455,
                      "end": 6457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6455,
                    "end": 6457
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6466,
                      "end": 6474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6466,
                    "end": 6474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6483,
                      "end": 6494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6483,
                    "end": 6494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "link",
                    "loc": {
                      "start": 6503,
                      "end": 6507
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6503,
                    "end": 6507
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6516,
                      "end": 6520
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6516,
                    "end": 6520
                  }
                }
              ],
              "loc": {
                "start": 6445,
                "end": 6526
              }
            },
            "loc": {
              "start": 6432,
              "end": 6526
            }
          }
        ],
        "loc": {
          "start": 3044,
          "end": 6528
        }
      },
      "loc": {
        "start": 3035,
        "end": 6528
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 6529,
          "end": 6540
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
                "start": 6547,
                "end": 6549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6547,
              "end": 6549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6554,
                "end": 6563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6554,
              "end": 6563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6568,
                "end": 6587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6568,
              "end": 6587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6592,
                "end": 6607
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6592,
              "end": 6607
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 6612,
                "end": 6621
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6612,
              "end": 6621
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 6626,
                "end": 6637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6626,
              "end": 6637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 6642,
                "end": 6653
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6642,
              "end": 6653
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6658,
                "end": 6662
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6658,
              "end": 6662
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 6667,
                "end": 6673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6667,
              "end": 6673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 6678,
                "end": 6688
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6678,
              "end": 6688
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "team",
              "loc": {
                "start": 6693,
                "end": 6697
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
                    "value": "Team_nav",
                    "loc": {
                      "start": 6711,
                      "end": 6719
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6708,
                    "end": 6719
                  }
                }
              ],
              "loc": {
                "start": 6698,
                "end": 6725
              }
            },
            "loc": {
              "start": 6693,
              "end": 6725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 6730,
                "end": 6734
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
                      "start": 6748,
                      "end": 6756
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6745,
                    "end": 6756
                  }
                }
              ],
              "loc": {
                "start": 6735,
                "end": 6762
              }
            },
            "loc": {
              "start": 6730,
              "end": 6762
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6767,
                "end": 6770
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
                      "start": 6781,
                      "end": 6790
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6781,
                    "end": 6790
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6799,
                      "end": 6808
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6799,
                    "end": 6808
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6817,
                      "end": 6824
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6817,
                    "end": 6824
                  }
                }
              ],
              "loc": {
                "start": 6771,
                "end": 6830
              }
            },
            "loc": {
              "start": 6767,
              "end": 6830
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "lastStep",
              "loc": {
                "start": 6835,
                "end": 6843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6835,
              "end": 6843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectVersion",
              "loc": {
                "start": 6848,
                "end": 6862
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
                      "start": 6873,
                      "end": 6875
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6873,
                    "end": 6875
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 6884,
                      "end": 6894
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6884,
                    "end": 6894
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6903,
                      "end": 6911
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6903,
                    "end": 6911
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6920,
                      "end": 6929
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6920,
                    "end": 6929
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6938,
                      "end": 6950
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6938,
                    "end": 6950
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6959,
                      "end": 6971
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6959,
                    "end": 6971
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6980,
                      "end": 6984
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
                            "start": 6999,
                            "end": 7001
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6999,
                          "end": 7001
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7014,
                            "end": 7023
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7014,
                          "end": 7023
                        }
                      }
                    ],
                    "loc": {
                      "start": 6985,
                      "end": 7033
                    }
                  },
                  "loc": {
                    "start": 6980,
                    "end": 7033
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 7042,
                      "end": 7054
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
                            "start": 7069,
                            "end": 7071
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7069,
                          "end": 7071
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 7084,
                            "end": 7092
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7084,
                          "end": 7092
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 7105,
                            "end": 7116
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7105,
                          "end": 7116
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7129,
                            "end": 7133
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7129,
                          "end": 7133
                        }
                      }
                    ],
                    "loc": {
                      "start": 7055,
                      "end": 7143
                    }
                  },
                  "loc": {
                    "start": 7042,
                    "end": 7143
                  }
                }
              ],
              "loc": {
                "start": 6863,
                "end": 7149
              }
            },
            "loc": {
              "start": 6848,
              "end": 7149
            }
          }
        ],
        "loc": {
          "start": 6541,
          "end": 7151
        }
      },
      "loc": {
        "start": 6529,
        "end": 7151
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 7152,
          "end": 7163
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
                "start": 7170,
                "end": 7172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7170,
              "end": 7172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7177,
                "end": 7186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7177,
              "end": 7186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 7191,
                "end": 7210
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7191,
              "end": 7210
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 7215,
                "end": 7230
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7215,
              "end": 7230
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 7235,
                "end": 7244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7235,
              "end": 7244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 7249,
                "end": 7260
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7249,
              "end": 7260
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7265,
                "end": 7276
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7265,
              "end": 7276
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7281,
                "end": 7285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7281,
              "end": 7285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7290,
                "end": 7296
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7290,
              "end": 7296
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 7301,
                "end": 7312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7301,
              "end": 7312
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 7317,
                "end": 7329
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7317,
              "end": 7329
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7334,
                "end": 7344
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7334,
              "end": 7344
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 7349,
                "end": 7368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7349,
              "end": 7368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "team",
              "loc": {
                "start": 7373,
                "end": 7377
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
                    "value": "Team_nav",
                    "loc": {
                      "start": 7391,
                      "end": 7399
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7388,
                    "end": 7399
                  }
                }
              ],
              "loc": {
                "start": 7378,
                "end": 7405
              }
            },
            "loc": {
              "start": 7373,
              "end": 7405
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 7410,
                "end": 7414
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
                      "start": 7428,
                      "end": 7436
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7425,
                    "end": 7436
                  }
                }
              ],
              "loc": {
                "start": 7415,
                "end": 7442
              }
            },
            "loc": {
              "start": 7410,
              "end": 7442
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7447,
                "end": 7450
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
                      "start": 7461,
                      "end": 7470
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7461,
                    "end": 7470
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7479,
                      "end": 7488
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7479,
                    "end": 7488
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7497,
                      "end": 7504
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7497,
                    "end": 7504
                  }
                }
              ],
              "loc": {
                "start": 7451,
                "end": 7510
              }
            },
            "loc": {
              "start": 7447,
              "end": 7510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "lastStep",
              "loc": {
                "start": 7515,
                "end": 7523
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7515,
              "end": 7523
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineVersion",
              "loc": {
                "start": 7528,
                "end": 7542
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
                      "start": 7553,
                      "end": 7555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7553,
                    "end": 7555
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 7564,
                      "end": 7574
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7564,
                    "end": 7574
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 7583,
                      "end": 7596
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7583,
                    "end": 7596
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 7605,
                      "end": 7615
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7605,
                    "end": 7615
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 7624,
                      "end": 7633
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7624,
                    "end": 7633
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 7642,
                      "end": 7650
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7642,
                    "end": 7650
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7659,
                      "end": 7668
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7659,
                    "end": 7668
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 7677,
                      "end": 7681
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
                            "start": 7696,
                            "end": 7698
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7696,
                          "end": 7698
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 7711,
                            "end": 7721
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7711,
                          "end": 7721
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7734,
                            "end": 7743
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7734,
                          "end": 7743
                        }
                      }
                    ],
                    "loc": {
                      "start": 7682,
                      "end": 7753
                    }
                  },
                  "loc": {
                    "start": 7677,
                    "end": 7753
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineType",
                    "loc": {
                      "start": 7762,
                      "end": 7773
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7762,
                    "end": 7773
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 7782,
                      "end": 7794
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
                            "start": 7809,
                            "end": 7811
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7809,
                          "end": 7811
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 7824,
                            "end": 7832
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7824,
                          "end": 7832
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 7845,
                            "end": 7856
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7845,
                          "end": 7856
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 7869,
                            "end": 7881
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7869,
                          "end": 7881
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7894,
                            "end": 7898
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7894,
                          "end": 7898
                        }
                      }
                    ],
                    "loc": {
                      "start": 7795,
                      "end": 7908
                    }
                  },
                  "loc": {
                    "start": 7782,
                    "end": 7908
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 7917,
                      "end": 7929
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7917,
                    "end": 7929
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 7938,
                      "end": 7950
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7938,
                    "end": 7950
                  }
                }
              ],
              "loc": {
                "start": 7543,
                "end": 7956
              }
            },
            "loc": {
              "start": 7528,
              "end": 7956
            }
          }
        ],
        "loc": {
          "start": 7164,
          "end": 7958
        }
      },
      "loc": {
        "start": 7152,
        "end": 7958
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7989,
          "end": 7991
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7989,
        "end": 7991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7992,
          "end": 8003
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7992,
        "end": 8003
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8004,
          "end": 8010
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8004,
        "end": 8010
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 8011,
          "end": 8023
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8011,
        "end": 8023
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 8024,
          "end": 8027
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
                "start": 8034,
                "end": 8047
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8034,
              "end": 8047
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 8052,
                "end": 8061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8052,
              "end": 8061
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 8066,
                "end": 8077
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8066,
              "end": 8077
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 8082,
                "end": 8091
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8082,
              "end": 8091
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 8096,
                "end": 8105
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8096,
              "end": 8105
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 8110,
                "end": 8117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8110,
              "end": 8117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 8122,
                "end": 8134
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8122,
              "end": 8134
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 8139,
                "end": 8147
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8139,
              "end": 8147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 8152,
                "end": 8166
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
                      "start": 8177,
                      "end": 8179
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8177,
                    "end": 8179
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 8188,
                      "end": 8198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8188,
                    "end": 8198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 8207,
                      "end": 8217
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8207,
                    "end": 8217
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 8226,
                      "end": 8233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8226,
                    "end": 8233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 8242,
                      "end": 8253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8242,
                    "end": 8253
                  }
                }
              ],
              "loc": {
                "start": 8167,
                "end": 8259
              }
            },
            "loc": {
              "start": 8152,
              "end": 8259
            }
          }
        ],
        "loc": {
          "start": 8028,
          "end": 8261
        }
      },
      "loc": {
        "start": 8024,
        "end": 8261
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8292,
          "end": 8294
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8292,
        "end": 8294
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8295,
          "end": 8305
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8295,
        "end": 8305
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 8306,
          "end": 8316
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8306,
        "end": 8316
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 8317,
          "end": 8328
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8317,
        "end": 8328
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8329,
          "end": 8335
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8329,
        "end": 8335
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 8336,
          "end": 8341
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8336,
        "end": 8341
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 8342,
          "end": 8362
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8342,
        "end": 8362
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8363,
          "end": 8367
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8363,
        "end": 8367
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 8368,
          "end": 8380
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8368,
        "end": 8380
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
                      "value": "Team",
                      "loc": {
                        "start": 88,
                        "end": 92
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 92
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
                          "value": "Team_nav",
                          "loc": {
                            "start": 106,
                            "end": 114
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 103,
                          "end": 114
                        }
                      }
                    ],
                    "loc": {
                      "start": 93,
                      "end": 120
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 120
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
                        "start": 132,
                        "end": 136
                      }
                    },
                    "loc": {
                      "start": 132,
                      "end": 136
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
                            "start": 150,
                            "end": 158
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 147,
                          "end": 158
                        }
                      }
                    ],
                    "loc": {
                      "start": 137,
                      "end": 164
                    }
                  },
                  "loc": {
                    "start": 125,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 166
              }
            },
            "loc": {
              "start": 69,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 167,
                "end": 170
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
                      "start": 177,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 177,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 191,
                      "end": 200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 200
                  }
                }
              ],
              "loc": {
                "start": 171,
                "end": 202
              }
            },
            "loc": {
              "start": 167,
              "end": 202
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 204
        }
      },
      "loc": {
        "start": 1,
        "end": 204
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 214,
          "end": 227
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 231,
            "end": 239
          }
        },
        "loc": {
          "start": 231,
          "end": 239
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
                "start": 242,
                "end": 244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 242,
              "end": 244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 245,
                "end": 255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 245,
              "end": 255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 256,
                "end": 266
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 256,
              "end": 266
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 267,
                "end": 271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 267,
              "end": 271
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 272,
                "end": 283
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 272,
              "end": 283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 284,
                "end": 291
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 284,
              "end": 291
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 292,
                "end": 297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 292,
              "end": 297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 298,
                "end": 308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 298,
              "end": 308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 309,
                "end": 322
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
                      "start": 329,
                      "end": 331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 329,
                    "end": 331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 336,
                      "end": 346
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 336,
                    "end": 346
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 351,
                      "end": 361
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 351,
                    "end": 361
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 366,
                      "end": 370
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 366,
                    "end": 370
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 375,
                      "end": 386
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 375,
                    "end": 386
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 391,
                      "end": 398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 403,
                      "end": 408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 403,
                    "end": 408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 413,
                      "end": 423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 413,
                    "end": 423
                  }
                }
              ],
              "loc": {
                "start": 323,
                "end": 425
              }
            },
            "loc": {
              "start": 309,
              "end": 425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 426,
                "end": 438
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
                      "start": 445,
                      "end": 447
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 445,
                    "end": 447
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 452,
                      "end": 462
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 452,
                    "end": 462
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 467,
                      "end": 477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 467,
                    "end": 477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 482,
                      "end": 491
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
                            "start": 502,
                            "end": 504
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 502,
                          "end": 504
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 513,
                            "end": 517
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 513,
                          "end": 517
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 526,
                            "end": 537
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 526,
                          "end": 537
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 546,
                            "end": 549
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
                                  "start": 564,
                                  "end": 573
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 564,
                                "end": 573
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 586,
                                  "end": 593
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 586,
                                "end": 593
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 606,
                                  "end": 615
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 606,
                                "end": 615
                              }
                            }
                          ],
                          "loc": {
                            "start": 550,
                            "end": 625
                          }
                        },
                        "loc": {
                          "start": 546,
                          "end": 625
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 634,
                            "end": 640
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
                                  "start": 655,
                                  "end": 657
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 655,
                                "end": 657
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 670,
                                  "end": 675
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 670,
                                "end": 675
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 688,
                                  "end": 693
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 688,
                                "end": 693
                              }
                            }
                          ],
                          "loc": {
                            "start": 641,
                            "end": 703
                          }
                        },
                        "loc": {
                          "start": 634,
                          "end": 703
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 712,
                            "end": 724
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
                                  "start": 739,
                                  "end": 741
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 739,
                                "end": 741
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 754,
                                  "end": 764
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 754,
                                "end": 764
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 777,
                                  "end": 789
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
                                        "start": 808,
                                        "end": 810
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 808,
                                      "end": 810
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 827,
                                        "end": 835
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 827,
                                      "end": 835
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 852,
                                        "end": 863
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 852,
                                      "end": 863
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 880,
                                        "end": 884
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 880,
                                      "end": 884
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 790,
                                  "end": 898
                                }
                              },
                              "loc": {
                                "start": 777,
                                "end": 898
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 911,
                                  "end": 920
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
                                        "start": 939,
                                        "end": 941
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 939,
                                      "end": 941
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 958,
                                        "end": 963
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 958,
                                      "end": 963
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 980,
                                        "end": 984
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 980,
                                      "end": 984
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 1001,
                                        "end": 1008
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1001,
                                      "end": 1008
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 1025,
                                        "end": 1037
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
                                              "start": 1060,
                                              "end": 1062
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1060,
                                            "end": 1062
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 1083,
                                              "end": 1091
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1083,
                                            "end": 1091
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1112,
                                              "end": 1123
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1112,
                                            "end": 1123
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1144,
                                              "end": 1148
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1144,
                                            "end": 1148
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1038,
                                        "end": 1166
                                      }
                                    },
                                    "loc": {
                                      "start": 1025,
                                      "end": 1166
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 921,
                                  "end": 1180
                                }
                              },
                              "loc": {
                                "start": 911,
                                "end": 1180
                              }
                            }
                          ],
                          "loc": {
                            "start": 725,
                            "end": 1190
                          }
                        },
                        "loc": {
                          "start": 712,
                          "end": 1190
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 1199,
                            "end": 1207
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
                                  "start": 1225,
                                  "end": 1240
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1222,
                                "end": 1240
                              }
                            }
                          ],
                          "loc": {
                            "start": 1208,
                            "end": 1250
                          }
                        },
                        "loc": {
                          "start": 1199,
                          "end": 1250
                        }
                      }
                    ],
                    "loc": {
                      "start": 492,
                      "end": 1256
                    }
                  },
                  "loc": {
                    "start": 482,
                    "end": 1256
                  }
                }
              ],
              "loc": {
                "start": 439,
                "end": 1258
              }
            },
            "loc": {
              "start": 426,
              "end": 1258
            }
          }
        ],
        "loc": {
          "start": 240,
          "end": 1260
        }
      },
      "loc": {
        "start": 205,
        "end": 1260
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1270,
          "end": 1283
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1287,
            "end": 1295
          }
        },
        "loc": {
          "start": 1287,
          "end": 1295
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
                "start": 1298,
                "end": 1300
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1298,
              "end": 1300
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1301,
                "end": 1306
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1301,
              "end": 1306
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1307,
                "end": 1311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1307,
              "end": 1311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1312,
                "end": 1319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1312,
              "end": 1319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1320,
                "end": 1332
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
                      "start": 1339,
                      "end": 1341
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1339,
                    "end": 1341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1346,
                      "end": 1354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1346,
                    "end": 1354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1359,
                      "end": 1370
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1359,
                    "end": 1370
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1375,
                      "end": 1379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1375,
                    "end": 1379
                  }
                }
              ],
              "loc": {
                "start": 1333,
                "end": 1381
              }
            },
            "loc": {
              "start": 1320,
              "end": 1381
            }
          }
        ],
        "loc": {
          "start": 1296,
          "end": 1383
        }
      },
      "loc": {
        "start": 1261,
        "end": 1383
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 1393,
          "end": 1408
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1412,
            "end": 1420
          }
        },
        "loc": {
          "start": 1412,
          "end": 1420
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
                "start": 1423,
                "end": 1425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1423,
              "end": 1425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1426,
                "end": 1436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1426,
              "end": 1436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1437,
                "end": 1447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1437,
              "end": 1447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1448,
                "end": 1457
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1448,
              "end": 1457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 1458,
                "end": 1465
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1458,
              "end": 1465
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 1466,
                "end": 1474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1466,
              "end": 1474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 1475,
                "end": 1485
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
                      "start": 1492,
                      "end": 1494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1492,
                    "end": 1494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 1499,
                      "end": 1516
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1499,
                    "end": 1516
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 1521,
                      "end": 1533
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1521,
                    "end": 1533
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 1538,
                      "end": 1548
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1538,
                    "end": 1548
                  }
                }
              ],
              "loc": {
                "start": 1486,
                "end": 1550
              }
            },
            "loc": {
              "start": 1475,
              "end": 1550
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 1551,
                "end": 1562
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
                      "start": 1569,
                      "end": 1571
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1569,
                    "end": 1571
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 1576,
                      "end": 1590
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1576,
                    "end": 1590
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 1595,
                      "end": 1603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1595,
                    "end": 1603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 1608,
                      "end": 1617
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1608,
                    "end": 1617
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 1622,
                      "end": 1632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1622,
                    "end": 1632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 1637,
                      "end": 1642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1637,
                    "end": 1642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
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
                }
              ],
              "loc": {
                "start": 1563,
                "end": 1656
              }
            },
            "loc": {
              "start": 1551,
              "end": 1656
            }
          }
        ],
        "loc": {
          "start": 1421,
          "end": 1658
        }
      },
      "loc": {
        "start": 1384,
        "end": 1658
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 1668,
          "end": 1681
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1685,
            "end": 1693
          }
        },
        "loc": {
          "start": 1685,
          "end": 1693
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
                "start": 1696,
                "end": 1698
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1696,
              "end": 1698
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1699,
                "end": 1709
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1699,
              "end": 1709
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1710,
                "end": 1720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1710,
              "end": 1720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1721,
                "end": 1730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1721,
              "end": 1730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 1731,
                "end": 1738
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1731,
              "end": 1738
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 1739,
                "end": 1747
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1739,
              "end": 1747
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 1748,
                "end": 1758
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
                      "start": 1765,
                      "end": 1767
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1765,
                    "end": 1767
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 1772,
                      "end": 1789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1772,
                    "end": 1789
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 1794,
                      "end": 1806
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1794,
                    "end": 1806
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 1811,
                      "end": 1821
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1811,
                    "end": 1821
                  }
                }
              ],
              "loc": {
                "start": 1759,
                "end": 1823
              }
            },
            "loc": {
              "start": 1748,
              "end": 1823
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 1824,
                "end": 1835
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
                      "start": 1842,
                      "end": 1844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1842,
                    "end": 1844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 1849,
                      "end": 1863
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1849,
                    "end": 1863
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 1868,
                      "end": 1876
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1868,
                    "end": 1876
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 1881,
                      "end": 1890
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1881,
                    "end": 1890
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 1895,
                      "end": 1905
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1895,
                    "end": 1905
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 1910,
                      "end": 1915
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1910,
                    "end": 1915
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 1920,
                      "end": 1927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1920,
                    "end": 1927
                  }
                }
              ],
              "loc": {
                "start": 1836,
                "end": 1929
              }
            },
            "loc": {
              "start": 1824,
              "end": 1929
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1930,
                "end": 1936
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
                      "start": 1946,
                      "end": 1956
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1943,
                    "end": 1956
                  }
                }
              ],
              "loc": {
                "start": 1937,
                "end": 1958
              }
            },
            "loc": {
              "start": 1930,
              "end": 1958
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 1959,
                "end": 1969
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
                      "start": 1976,
                      "end": 1978
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1976,
                    "end": 1978
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1983,
                      "end": 1987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1983,
                    "end": 1987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1992,
                      "end": 2003
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1992,
                    "end": 2003
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2008,
                      "end": 2011
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
                            "start": 2022,
                            "end": 2031
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2022,
                          "end": 2031
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2040,
                            "end": 2047
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2040,
                          "end": 2047
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2056,
                            "end": 2065
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2056,
                          "end": 2065
                        }
                      }
                    ],
                    "loc": {
                      "start": 2012,
                      "end": 2071
                    }
                  },
                  "loc": {
                    "start": 2008,
                    "end": 2071
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 2076,
                      "end": 2082
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
                            "start": 2093,
                            "end": 2095
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2093,
                          "end": 2095
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 2104,
                            "end": 2109
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2104,
                          "end": 2109
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2118,
                            "end": 2123
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2118,
                          "end": 2123
                        }
                      }
                    ],
                    "loc": {
                      "start": 2083,
                      "end": 2129
                    }
                  },
                  "loc": {
                    "start": 2076,
                    "end": 2129
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminderList",
                    "loc": {
                      "start": 2134,
                      "end": 2146
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
                            "start": 2157,
                            "end": 2159
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2157,
                          "end": 2159
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2168,
                            "end": 2178
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2168,
                          "end": 2178
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2187,
                            "end": 2197
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2187,
                          "end": 2197
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminders",
                          "loc": {
                            "start": 2206,
                            "end": 2215
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
                                  "start": 2230,
                                  "end": 2232
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2230,
                                "end": 2232
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2245,
                                  "end": 2255
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2245,
                                "end": 2255
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2268,
                                  "end": 2278
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2268,
                                "end": 2278
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2291,
                                  "end": 2295
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2291,
                                "end": 2295
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2308,
                                  "end": 2319
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2308,
                                "end": 2319
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 2332,
                                  "end": 2339
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2332,
                                "end": 2339
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2352,
                                  "end": 2357
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2352,
                                "end": 2357
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 2370,
                                  "end": 2380
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2370,
                                "end": 2380
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderItems",
                                "loc": {
                                  "start": 2393,
                                  "end": 2406
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
                                        "start": 2425,
                                        "end": 2427
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2425,
                                      "end": 2427
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2444,
                                        "end": 2454
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2444,
                                      "end": 2454
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 2471,
                                        "end": 2481
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2471,
                                      "end": 2481
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2498,
                                        "end": 2502
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2498,
                                      "end": 2502
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2519,
                                        "end": 2530
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2519,
                                      "end": 2530
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 2547,
                                        "end": 2554
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2547,
                                      "end": 2554
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 2571,
                                        "end": 2576
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2571,
                                      "end": 2576
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 2593,
                                        "end": 2603
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2593,
                                      "end": 2603
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2407,
                                  "end": 2617
                                }
                              },
                              "loc": {
                                "start": 2393,
                                "end": 2617
                              }
                            }
                          ],
                          "loc": {
                            "start": 2216,
                            "end": 2627
                          }
                        },
                        "loc": {
                          "start": 2206,
                          "end": 2627
                        }
                      }
                    ],
                    "loc": {
                      "start": 2147,
                      "end": 2633
                    }
                  },
                  "loc": {
                    "start": 2134,
                    "end": 2633
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 2638,
                      "end": 2650
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
                            "start": 2661,
                            "end": 2663
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2661,
                          "end": 2663
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2672,
                            "end": 2682
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2672,
                          "end": 2682
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 2691,
                            "end": 2703
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
                                  "start": 2718,
                                  "end": 2720
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2718,
                                "end": 2720
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 2733,
                                  "end": 2741
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2733,
                                "end": 2741
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2754,
                                  "end": 2765
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2754,
                                "end": 2765
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2778,
                                  "end": 2782
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2778,
                                "end": 2782
                              }
                            }
                          ],
                          "loc": {
                            "start": 2704,
                            "end": 2792
                          }
                        },
                        "loc": {
                          "start": 2691,
                          "end": 2792
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 2801,
                            "end": 2810
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
                                  "start": 2825,
                                  "end": 2827
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2825,
                                "end": 2827
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2840,
                                  "end": 2845
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2840,
                                "end": 2845
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 2858,
                                  "end": 2862
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2858,
                                "end": 2862
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 2875,
                                  "end": 2882
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2875,
                                "end": 2882
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 2895,
                                  "end": 2907
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
                                        "start": 2926,
                                        "end": 2928
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2926,
                                      "end": 2928
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 2945,
                                        "end": 2953
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2945,
                                      "end": 2953
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2970,
                                        "end": 2981
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2970,
                                      "end": 2981
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2998,
                                        "end": 3002
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2998,
                                      "end": 3002
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2908,
                                  "end": 3016
                                }
                              },
                              "loc": {
                                "start": 2895,
                                "end": 3016
                              }
                            }
                          ],
                          "loc": {
                            "start": 2811,
                            "end": 3026
                          }
                        },
                        "loc": {
                          "start": 2801,
                          "end": 3026
                        }
                      }
                    ],
                    "loc": {
                      "start": 2651,
                      "end": 3032
                    }
                  },
                  "loc": {
                    "start": 2638,
                    "end": 3032
                  }
                }
              ],
              "loc": {
                "start": 1970,
                "end": 3034
              }
            },
            "loc": {
              "start": 1959,
              "end": 3034
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetings",
              "loc": {
                "start": 3035,
                "end": 3043
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
                      "start": 3050,
                      "end": 3052
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3050,
                    "end": 3052
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3057,
                      "end": 3067
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3057,
                    "end": 3067
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3072,
                      "end": 3082
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3072,
                    "end": 3082
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 3087,
                      "end": 3109
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3087,
                    "end": 3109
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnTeamProfile",
                    "loc": {
                      "start": 3114,
                      "end": 3131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3114,
                    "end": 3131
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "team",
                    "loc": {
                      "start": 3136,
                      "end": 3140
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
                            "start": 3151,
                            "end": 3153
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3151,
                          "end": 3153
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 3162,
                            "end": 3173
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3162,
                          "end": 3173
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 3182,
                            "end": 3188
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3182,
                          "end": 3188
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 3197,
                            "end": 3209
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3197,
                          "end": 3209
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 3218,
                            "end": 3221
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
                                  "start": 3236,
                                  "end": 3249
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3236,
                                "end": 3249
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 3262,
                                  "end": 3271
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3262,
                                "end": 3271
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 3284,
                                  "end": 3295
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3284,
                                "end": 3295
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 3308,
                                  "end": 3317
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3308,
                                "end": 3317
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 3330,
                                  "end": 3339
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3330,
                                "end": 3339
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 3352,
                                  "end": 3359
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3352,
                                "end": 3359
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 3372,
                                  "end": 3384
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3372,
                                "end": 3384
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 3397,
                                  "end": 3405
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3397,
                                "end": 3405
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 3418,
                                  "end": 3432
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
                                        "start": 3451,
                                        "end": 3453
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3451,
                                      "end": 3453
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3470,
                                        "end": 3480
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3470,
                                      "end": 3480
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3497,
                                        "end": 3507
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3497,
                                      "end": 3507
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 3524,
                                        "end": 3531
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3524,
                                      "end": 3531
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3548,
                                        "end": 3559
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3548,
                                      "end": 3559
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3433,
                                  "end": 3573
                                }
                              },
                              "loc": {
                                "start": 3418,
                                "end": 3573
                              }
                            }
                          ],
                          "loc": {
                            "start": 3222,
                            "end": 3583
                          }
                        },
                        "loc": {
                          "start": 3218,
                          "end": 3583
                        }
                      }
                    ],
                    "loc": {
                      "start": 3141,
                      "end": 3589
                    }
                  },
                  "loc": {
                    "start": 3136,
                    "end": 3589
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 3594,
                      "end": 3611
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
                            "start": 3622,
                            "end": 3624
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3622,
                          "end": 3624
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3633,
                            "end": 3643
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3633,
                          "end": 3643
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3652,
                            "end": 3662
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3652,
                          "end": 3662
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3671,
                            "end": 3675
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3671,
                          "end": 3675
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3684,
                            "end": 3695
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3684,
                          "end": 3695
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 3704,
                            "end": 3716
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3704,
                          "end": 3716
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "team",
                          "loc": {
                            "start": 3725,
                            "end": 3729
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
                                  "start": 3744,
                                  "end": 3746
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3744,
                                "end": 3746
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 3759,
                                  "end": 3770
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3759,
                                "end": 3770
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 3783,
                                  "end": 3789
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3783,
                                "end": 3789
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 3802,
                                  "end": 3814
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3802,
                                "end": 3814
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 3827,
                                  "end": 3830
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
                                        "start": 3849,
                                        "end": 3862
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3849,
                                      "end": 3862
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 3879,
                                        "end": 3888
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3879,
                                      "end": 3888
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 3905,
                                        "end": 3916
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3905,
                                      "end": 3916
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 3933,
                                        "end": 3942
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3933,
                                      "end": 3942
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 3959,
                                        "end": 3968
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3959,
                                      "end": 3968
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 3985,
                                        "end": 3992
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3985,
                                      "end": 3992
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 4009,
                                        "end": 4021
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4009,
                                      "end": 4021
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 4038,
                                        "end": 4046
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4038,
                                      "end": 4046
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 4063,
                                        "end": 4077
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
                                              "start": 4100,
                                              "end": 4102
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4100,
                                            "end": 4102
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4123,
                                              "end": 4133
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4123,
                                            "end": 4133
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4154,
                                              "end": 4164
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4154,
                                            "end": 4164
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 4185,
                                              "end": 4192
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4185,
                                            "end": 4192
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 4213,
                                              "end": 4224
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4213,
                                            "end": 4224
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4078,
                                        "end": 4242
                                      }
                                    },
                                    "loc": {
                                      "start": 4063,
                                      "end": 4242
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3831,
                                  "end": 4256
                                }
                              },
                              "loc": {
                                "start": 3827,
                                "end": 4256
                              }
                            }
                          ],
                          "loc": {
                            "start": 3730,
                            "end": 4266
                          }
                        },
                        "loc": {
                          "start": 3725,
                          "end": 4266
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 4275,
                            "end": 4287
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
                                  "start": 4302,
                                  "end": 4304
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4302,
                                "end": 4304
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 4317,
                                  "end": 4325
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4317,
                                "end": 4325
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 4338,
                                  "end": 4349
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4338,
                                "end": 4349
                              }
                            }
                          ],
                          "loc": {
                            "start": 4288,
                            "end": 4359
                          }
                        },
                        "loc": {
                          "start": 4275,
                          "end": 4359
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "members",
                          "loc": {
                            "start": 4368,
                            "end": 4375
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
                                  "start": 4390,
                                  "end": 4392
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4390,
                                "end": 4392
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4405,
                                  "end": 4415
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4405,
                                "end": 4415
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4428,
                                  "end": 4438
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4428,
                                "end": 4438
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 4451,
                                  "end": 4458
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4451,
                                "end": 4458
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4471,
                                  "end": 4482
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4471,
                                "end": 4482
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 4495,
                                  "end": 4500
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
                                        "start": 4519,
                                        "end": 4521
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4519,
                                      "end": 4521
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4538,
                                        "end": 4548
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4538,
                                      "end": 4548
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4565,
                                        "end": 4575
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4565,
                                      "end": 4575
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4592,
                                        "end": 4596
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4592,
                                      "end": 4596
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4613,
                                        "end": 4624
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4613,
                                      "end": 4624
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 4641,
                                        "end": 4653
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4641,
                                      "end": 4653
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "team",
                                      "loc": {
                                        "start": 4670,
                                        "end": 4674
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
                                              "start": 4697,
                                              "end": 4699
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4697,
                                            "end": 4699
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 4720,
                                              "end": 4731
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4720,
                                            "end": 4731
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 4752,
                                              "end": 4758
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4752,
                                            "end": 4758
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 4779,
                                              "end": 4791
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4779,
                                            "end": 4791
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 4812,
                                              "end": 4815
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
                                                    "start": 4842,
                                                    "end": 4855
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4842,
                                                  "end": 4855
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 4880,
                                                    "end": 4889
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4880,
                                                  "end": 4889
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 4914,
                                                    "end": 4925
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4914,
                                                  "end": 4925
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 4950,
                                                    "end": 4959
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4950,
                                                  "end": 4959
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 4984,
                                                    "end": 4993
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4984,
                                                  "end": 4993
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 5018,
                                                    "end": 5025
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5018,
                                                  "end": 5025
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 5050,
                                                    "end": 5062
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5050,
                                                  "end": 5062
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 5087,
                                                    "end": 5095
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5087,
                                                  "end": 5095
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 5120,
                                                    "end": 5134
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
                                                          "start": 5165,
                                                          "end": 5167
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5165,
                                                        "end": 5167
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 5196,
                                                          "end": 5206
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5196,
                                                        "end": 5206
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 5235,
                                                          "end": 5245
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5235,
                                                        "end": 5245
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 5274,
                                                          "end": 5281
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5274,
                                                        "end": 5281
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 5310,
                                                          "end": 5321
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5310,
                                                        "end": 5321
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 5135,
                                                    "end": 5347
                                                  }
                                                },
                                                "loc": {
                                                  "start": 5120,
                                                  "end": 5347
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4816,
                                              "end": 5369
                                            }
                                          },
                                          "loc": {
                                            "start": 4812,
                                            "end": 5369
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4675,
                                        "end": 5387
                                      }
                                    },
                                    "loc": {
                                      "start": 4670,
                                      "end": 5387
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5404,
                                        "end": 5416
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
                                              "start": 5439,
                                              "end": 5441
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5439,
                                            "end": 5441
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5462,
                                              "end": 5470
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5462,
                                            "end": 5470
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5491,
                                              "end": 5502
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5491,
                                            "end": 5502
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5417,
                                        "end": 5520
                                      }
                                    },
                                    "loc": {
                                      "start": 5404,
                                      "end": 5520
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4501,
                                  "end": 5534
                                }
                              },
                              "loc": {
                                "start": 4495,
                                "end": 5534
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5547,
                                  "end": 5550
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
                                        "start": 5569,
                                        "end": 5578
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5569,
                                      "end": 5578
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 5595,
                                        "end": 5604
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5595,
                                      "end": 5604
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5551,
                                  "end": 5618
                                }
                              },
                              "loc": {
                                "start": 5547,
                                "end": 5618
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "user",
                                "loc": {
                                  "start": 5631,
                                  "end": 5635
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
                                        "start": 5654,
                                        "end": 5656
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5654,
                                      "end": 5656
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5673,
                                        "end": 5683
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5673,
                                      "end": 5683
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 5700,
                                        "end": 5710
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5700,
                                      "end": 5710
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 5727,
                                        "end": 5738
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5727,
                                      "end": 5738
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 5755,
                                        "end": 5761
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5755,
                                      "end": 5761
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBot",
                                      "loc": {
                                        "start": 5778,
                                        "end": 5783
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5778,
                                      "end": 5783
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBotDepictingPerson",
                                      "loc": {
                                        "start": 5800,
                                        "end": 5820
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5800,
                                      "end": 5820
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 5837,
                                        "end": 5841
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5837,
                                      "end": 5841
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 5858,
                                        "end": 5870
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5858,
                                      "end": 5870
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 5887,
                                        "end": 5896
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5887,
                                      "end": 5896
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reportsReceivedCount",
                                      "loc": {
                                        "start": 5913,
                                        "end": 5933
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5913,
                                      "end": 5933
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5950,
                                        "end": 5953
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
                                              "start": 5976,
                                              "end": 5985
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5976,
                                            "end": 5985
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 6006,
                                              "end": 6015
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6006,
                                            "end": 6015
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 6036,
                                              "end": 6045
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6036,
                                            "end": 6045
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 6066,
                                              "end": 6078
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6066,
                                            "end": 6078
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 6099,
                                              "end": 6107
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6099,
                                            "end": 6107
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5954,
                                        "end": 6125
                                      }
                                    },
                                    "loc": {
                                      "start": 5950,
                                      "end": 6125
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 6142,
                                        "end": 6154
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
                                              "start": 6177,
                                              "end": 6179
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6177,
                                            "end": 6179
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 6200,
                                              "end": 6208
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6200,
                                            "end": 6208
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bio",
                                            "loc": {
                                              "start": 6229,
                                              "end": 6232
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6229,
                                            "end": 6232
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6155,
                                        "end": 6250
                                      }
                                    },
                                    "loc": {
                                      "start": 6142,
                                      "end": 6250
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5636,
                                  "end": 6264
                                }
                              },
                              "loc": {
                                "start": 5631,
                                "end": 6264
                              }
                            }
                          ],
                          "loc": {
                            "start": 4376,
                            "end": 6274
                          }
                        },
                        "loc": {
                          "start": 4368,
                          "end": 6274
                        }
                      }
                    ],
                    "loc": {
                      "start": 3612,
                      "end": 6280
                    }
                  },
                  "loc": {
                    "start": 3594,
                    "end": 6280
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 6285,
                      "end": 6299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6285,
                    "end": 6299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 6304,
                      "end": 6316
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6304,
                    "end": 6316
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6321,
                      "end": 6324
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
                            "start": 6335,
                            "end": 6344
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6335,
                          "end": 6344
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 6353,
                            "end": 6362
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6353,
                          "end": 6362
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6371,
                            "end": 6380
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6371,
                          "end": 6380
                        }
                      }
                    ],
                    "loc": {
                      "start": 6325,
                      "end": 6386
                    }
                  },
                  "loc": {
                    "start": 6321,
                    "end": 6386
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 6391,
                      "end": 6397
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
                            "start": 6411,
                            "end": 6421
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6408,
                          "end": 6421
                        }
                      }
                    ],
                    "loc": {
                      "start": 6398,
                      "end": 6427
                    }
                  },
                  "loc": {
                    "start": 6391,
                    "end": 6427
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6432,
                      "end": 6444
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
                            "start": 6455,
                            "end": 6457
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6455,
                          "end": 6457
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6466,
                            "end": 6474
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6466,
                          "end": 6474
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6483,
                            "end": 6494
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6483,
                          "end": 6494
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 6503,
                            "end": 6507
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6503,
                          "end": 6507
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6516,
                            "end": 6520
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6516,
                          "end": 6520
                        }
                      }
                    ],
                    "loc": {
                      "start": 6445,
                      "end": 6526
                    }
                  },
                  "loc": {
                    "start": 6432,
                    "end": 6526
                  }
                }
              ],
              "loc": {
                "start": 3044,
                "end": 6528
              }
            },
            "loc": {
              "start": 3035,
              "end": 6528
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 6529,
                "end": 6540
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
                      "start": 6547,
                      "end": 6549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6547,
                    "end": 6549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6554,
                      "end": 6563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6554,
                    "end": 6563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6568,
                      "end": 6587
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6568,
                    "end": 6587
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6592,
                      "end": 6607
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6592,
                    "end": 6607
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 6612,
                      "end": 6621
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6612,
                    "end": 6621
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6626,
                      "end": 6637
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6626,
                    "end": 6637
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 6642,
                      "end": 6653
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6642,
                    "end": 6653
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6658,
                      "end": 6662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6658,
                    "end": 6662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 6667,
                      "end": 6673
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6667,
                    "end": 6673
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 6678,
                      "end": 6688
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6678,
                    "end": 6688
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "team",
                    "loc": {
                      "start": 6693,
                      "end": 6697
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
                          "value": "Team_nav",
                          "loc": {
                            "start": 6711,
                            "end": 6719
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6708,
                          "end": 6719
                        }
                      }
                    ],
                    "loc": {
                      "start": 6698,
                      "end": 6725
                    }
                  },
                  "loc": {
                    "start": 6693,
                    "end": 6725
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 6730,
                      "end": 6734
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
                            "start": 6748,
                            "end": 6756
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6745,
                          "end": 6756
                        }
                      }
                    ],
                    "loc": {
                      "start": 6735,
                      "end": 6762
                    }
                  },
                  "loc": {
                    "start": 6730,
                    "end": 6762
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6767,
                      "end": 6770
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
                            "start": 6781,
                            "end": 6790
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6781,
                          "end": 6790
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6799,
                            "end": 6808
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6799,
                          "end": 6808
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6817,
                            "end": 6824
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6817,
                          "end": 6824
                        }
                      }
                    ],
                    "loc": {
                      "start": 6771,
                      "end": 6830
                    }
                  },
                  "loc": {
                    "start": 6767,
                    "end": 6830
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "lastStep",
                    "loc": {
                      "start": 6835,
                      "end": 6843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6835,
                    "end": 6843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectVersion",
                    "loc": {
                      "start": 6848,
                      "end": 6862
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
                            "start": 6873,
                            "end": 6875
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6873,
                          "end": 6875
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 6884,
                            "end": 6894
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6884,
                          "end": 6894
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 6903,
                            "end": 6911
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6903,
                          "end": 6911
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6920,
                            "end": 6929
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6920,
                          "end": 6929
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6938,
                            "end": 6950
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6938,
                          "end": 6950
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6959,
                            "end": 6971
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6959,
                          "end": 6971
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6980,
                            "end": 6984
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
                                  "start": 6999,
                                  "end": 7001
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6999,
                                "end": 7001
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 7014,
                                  "end": 7023
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7014,
                                "end": 7023
                              }
                            }
                          ],
                          "loc": {
                            "start": 6985,
                            "end": 7033
                          }
                        },
                        "loc": {
                          "start": 6980,
                          "end": 7033
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 7042,
                            "end": 7054
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
                                  "start": 7069,
                                  "end": 7071
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7069,
                                "end": 7071
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 7084,
                                  "end": 7092
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7084,
                                "end": 7092
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7105,
                                  "end": 7116
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7105,
                                "end": 7116
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7129,
                                  "end": 7133
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7129,
                                "end": 7133
                              }
                            }
                          ],
                          "loc": {
                            "start": 7055,
                            "end": 7143
                          }
                        },
                        "loc": {
                          "start": 7042,
                          "end": 7143
                        }
                      }
                    ],
                    "loc": {
                      "start": 6863,
                      "end": 7149
                    }
                  },
                  "loc": {
                    "start": 6848,
                    "end": 7149
                  }
                }
              ],
              "loc": {
                "start": 6541,
                "end": 7151
              }
            },
            "loc": {
              "start": 6529,
              "end": 7151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 7152,
                "end": 7163
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
                      "start": 7170,
                      "end": 7172
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7170,
                    "end": 7172
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7177,
                      "end": 7186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7177,
                    "end": 7186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 7191,
                      "end": 7210
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7191,
                    "end": 7210
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 7215,
                      "end": 7230
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7215,
                    "end": 7230
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 7235,
                      "end": 7244
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7235,
                    "end": 7244
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 7249,
                      "end": 7260
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7249,
                    "end": 7260
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7265,
                      "end": 7276
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7265,
                    "end": 7276
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7281,
                      "end": 7285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7281,
                    "end": 7285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7290,
                      "end": 7296
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7290,
                    "end": 7296
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 7301,
                      "end": 7312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7301,
                    "end": 7312
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 7317,
                      "end": 7329
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7317,
                    "end": 7329
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7334,
                      "end": 7344
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7334,
                    "end": 7344
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 7349,
                      "end": 7368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7349,
                    "end": 7368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "team",
                    "loc": {
                      "start": 7373,
                      "end": 7377
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
                          "value": "Team_nav",
                          "loc": {
                            "start": 7391,
                            "end": 7399
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7388,
                          "end": 7399
                        }
                      }
                    ],
                    "loc": {
                      "start": 7378,
                      "end": 7405
                    }
                  },
                  "loc": {
                    "start": 7373,
                    "end": 7405
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 7410,
                      "end": 7414
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
                            "start": 7428,
                            "end": 7436
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7425,
                          "end": 7436
                        }
                      }
                    ],
                    "loc": {
                      "start": 7415,
                      "end": 7442
                    }
                  },
                  "loc": {
                    "start": 7410,
                    "end": 7442
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7447,
                      "end": 7450
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
                            "start": 7461,
                            "end": 7470
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7461,
                          "end": 7470
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7479,
                            "end": 7488
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7479,
                          "end": 7488
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7497,
                            "end": 7504
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7497,
                          "end": 7504
                        }
                      }
                    ],
                    "loc": {
                      "start": 7451,
                      "end": 7510
                    }
                  },
                  "loc": {
                    "start": 7447,
                    "end": 7510
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "lastStep",
                    "loc": {
                      "start": 7515,
                      "end": 7523
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7515,
                    "end": 7523
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineVersion",
                    "loc": {
                      "start": 7528,
                      "end": 7542
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
                            "start": 7553,
                            "end": 7555
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7553,
                          "end": 7555
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 7564,
                            "end": 7574
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7564,
                          "end": 7574
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 7583,
                            "end": 7596
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7583,
                          "end": 7596
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 7605,
                            "end": 7615
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7605,
                          "end": 7615
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 7624,
                            "end": 7633
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7624,
                          "end": 7633
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 7642,
                            "end": 7650
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7642,
                          "end": 7650
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7659,
                            "end": 7668
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7659,
                          "end": 7668
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 7677,
                            "end": 7681
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
                                  "start": 7696,
                                  "end": 7698
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7696,
                                "end": 7698
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 7711,
                                  "end": 7721
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7711,
                                "end": 7721
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 7734,
                                  "end": 7743
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7734,
                                "end": 7743
                              }
                            }
                          ],
                          "loc": {
                            "start": 7682,
                            "end": 7753
                          }
                        },
                        "loc": {
                          "start": 7677,
                          "end": 7753
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "routineType",
                          "loc": {
                            "start": 7762,
                            "end": 7773
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7762,
                          "end": 7773
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 7782,
                            "end": 7794
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
                                  "start": 7809,
                                  "end": 7811
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7809,
                                "end": 7811
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 7824,
                                  "end": 7832
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7824,
                                "end": 7832
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7845,
                                  "end": 7856
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7845,
                                "end": 7856
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 7869,
                                  "end": 7881
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7869,
                                "end": 7881
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7894,
                                  "end": 7898
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7894,
                                "end": 7898
                              }
                            }
                          ],
                          "loc": {
                            "start": 7795,
                            "end": 7908
                          }
                        },
                        "loc": {
                          "start": 7782,
                          "end": 7908
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 7917,
                            "end": 7929
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7917,
                          "end": 7929
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 7938,
                            "end": 7950
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7938,
                          "end": 7950
                        }
                      }
                    ],
                    "loc": {
                      "start": 7543,
                      "end": 7956
                    }
                  },
                  "loc": {
                    "start": 7528,
                    "end": 7956
                  }
                }
              ],
              "loc": {
                "start": 7164,
                "end": 7958
              }
            },
            "loc": {
              "start": 7152,
              "end": 7958
            }
          }
        ],
        "loc": {
          "start": 1694,
          "end": 7960
        }
      },
      "loc": {
        "start": 1659,
        "end": 7960
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 7970,
          "end": 7978
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 7982,
            "end": 7986
          }
        },
        "loc": {
          "start": 7982,
          "end": 7986
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
                "start": 7989,
                "end": 7991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7989,
              "end": 7991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7992,
                "end": 8003
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7992,
              "end": 8003
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8004,
                "end": 8010
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8004,
              "end": 8010
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 8011,
                "end": 8023
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8011,
              "end": 8023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8024,
                "end": 8027
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
                      "start": 8034,
                      "end": 8047
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8034,
                    "end": 8047
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 8052,
                      "end": 8061
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8052,
                    "end": 8061
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 8066,
                      "end": 8077
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8066,
                    "end": 8077
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 8082,
                      "end": 8091
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8082,
                    "end": 8091
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 8096,
                      "end": 8105
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8096,
                    "end": 8105
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 8110,
                      "end": 8117
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8110,
                    "end": 8117
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 8122,
                      "end": 8134
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8122,
                    "end": 8134
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 8139,
                      "end": 8147
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8139,
                    "end": 8147
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 8152,
                      "end": 8166
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
                            "start": 8177,
                            "end": 8179
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8177,
                          "end": 8179
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 8188,
                            "end": 8198
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8188,
                          "end": 8198
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 8207,
                            "end": 8217
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8207,
                          "end": 8217
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 8226,
                            "end": 8233
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8226,
                          "end": 8233
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 8242,
                            "end": 8253
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8242,
                          "end": 8253
                        }
                      }
                    ],
                    "loc": {
                      "start": 8167,
                      "end": 8259
                    }
                  },
                  "loc": {
                    "start": 8152,
                    "end": 8259
                  }
                }
              ],
              "loc": {
                "start": 8028,
                "end": 8261
              }
            },
            "loc": {
              "start": 8024,
              "end": 8261
            }
          }
        ],
        "loc": {
          "start": 7987,
          "end": 8263
        }
      },
      "loc": {
        "start": 7961,
        "end": 8263
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 8273,
          "end": 8281
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8285,
            "end": 8289
          }
        },
        "loc": {
          "start": 8285,
          "end": 8289
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
                "start": 8292,
                "end": 8294
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8292,
              "end": 8294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8295,
                "end": 8305
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8295,
              "end": 8305
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 8306,
                "end": 8316
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8306,
              "end": 8316
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 8317,
                "end": 8328
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8317,
              "end": 8328
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8329,
                "end": 8335
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8329,
              "end": 8335
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8336,
                "end": 8341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8336,
              "end": 8341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 8342,
                "end": 8362
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8342,
              "end": 8362
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8363,
                "end": 8367
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8363,
              "end": 8367
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 8368,
                "end": 8380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8368,
              "end": 8380
            }
          }
        ],
        "loc": {
          "start": 8290,
          "end": 8382
        }
      },
      "loc": {
        "start": 8264,
        "end": 8382
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
        "start": 8390,
        "end": 8394
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
              "start": 8396,
              "end": 8401
            }
          },
          "loc": {
            "start": 8395,
            "end": 8401
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
                "start": 8403,
                "end": 8412
              }
            },
            "loc": {
              "start": 8403,
              "end": 8412
            }
          },
          "loc": {
            "start": 8403,
            "end": 8413
          }
        },
        "directives": [],
        "loc": {
          "start": 8395,
          "end": 8413
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
              "start": 8419,
              "end": 8423
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8424,
                  "end": 8429
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8432,
                    "end": 8437
                  }
                },
                "loc": {
                  "start": 8431,
                  "end": 8437
                }
              },
              "loc": {
                "start": 8424,
                "end": 8437
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
                    "start": 8445,
                    "end": 8456
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
                          "start": 8470,
                          "end": 8483
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8467,
                        "end": 8483
                      }
                    }
                  ],
                  "loc": {
                    "start": 8457,
                    "end": 8489
                  }
                },
                "loc": {
                  "start": 8445,
                  "end": 8489
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 8494,
                    "end": 8503
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
                          "start": 8517,
                          "end": 8530
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8514,
                        "end": 8530
                      }
                    }
                  ],
                  "loc": {
                    "start": 8504,
                    "end": 8536
                  }
                },
                "loc": {
                  "start": 8494,
                  "end": 8536
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 8541,
                    "end": 8550
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
                          "start": 8564,
                          "end": 8577
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8561,
                        "end": 8577
                      }
                    }
                  ],
                  "loc": {
                    "start": 8551,
                    "end": 8583
                  }
                },
                "loc": {
                  "start": 8541,
                  "end": 8583
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 8588,
                    "end": 8597
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
                          "start": 8611,
                          "end": 8624
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8608,
                        "end": 8624
                      }
                    }
                  ],
                  "loc": {
                    "start": 8598,
                    "end": 8630
                  }
                },
                "loc": {
                  "start": 8588,
                  "end": 8630
                }
              }
            ],
            "loc": {
              "start": 8439,
              "end": 8634
            }
          },
          "loc": {
            "start": 8419,
            "end": 8634
          }
        }
      ],
      "loc": {
        "start": 8415,
        "end": 8636
      }
    },
    "loc": {
      "start": 8384,
      "end": 8636
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
